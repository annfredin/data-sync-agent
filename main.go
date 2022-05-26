package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"time"

	_ "github.com/lib/pq"

	"data-sync-agent/config"
	"data-sync-agent/helper"
	"data-sync-agent/model"
	"data-sync-agent/utils/logger"

	"data-sync-agent/crypto"
	"data-sync-agent/dataservice/postgreprovider"
	"data-sync-agent/dataservice/sqldataprovider"

	"data-sync-agent/entity"
)

var sqlConnectionListData []*model.SQLConnectionData
var allowedMainPageIDs string
var updSysncDateErrorCount int
var isAppAlive = true

//default values..
var redisKeyForRegisteredDevice, redisKeyForTestDevice,
	redisKeyForCommunicationGroup, redisKeyForDeviceCommandChannel string
var singlePartitionDeviceCount int

//WorkerResponse from each worker
type WorkerResponse struct {
	task                        *model.SQLConnectionData
	registeredDeviceRequestData *model.RegisteredDeviceRequestData
	spatialRequestData          *model.SpatialRequestData
}

//WorkerPoolResponse has consolidated result collected from all the workers
type WorkerPoolResponse struct {
	registeredDeviceDataList []*model.RegisteredDeviceData
	spatialRequestData       *model.SpatialRequestData
	dataFetchRequestData     map[string]model.DataFetchRequestData
}

//entry for app..
func main() {
	// bootstrap app!!!!
	go func() {
		initDataSyncJob()
		prepareJob()
	}()

	//close app
	closeAppSetup()
}

func initDataSyncJob() {
	logger.Log().Info("Initialization Start - V1 !!")

	redisKeyForOnboardedSQLServers := helper.GetEnv(helper.RedisKeyForOnBoardedServers)

	if redisKeyForOnboardedSQLServers == "" {
		logger.Log().Error("startDataSyncJob - redisKeyForOnboardedSQLServers is empty")
		return
	}

	redisKeyForSQLServers := helper.GetEnv(helper.RedisKeyForServers)

	if redisKeyForSQLServers == "" {
		logger.Log().Error("startDataSyncJob - redisKeyForSQLServers is empty")
		return
	}

	//initialize the crypto module..
	crypto.InitializeCryptoProvider(logger.Log())

	onboardedServerIds, err := config.SMembers(redisKeyForOnboardedSQLServers)
	if err != nil {
		logger.Log().Error(fmt.Sprintf(" startDataSyncJob Redis SQL On-boarded Servers Error : %v", err.Error()))
		return
	}
	serverValues, err := config.HGetAll(redisKeyForSQLServers)
	if err != nil {
		logger.Log().Error(fmt.Sprintf(" startDataSyncJob Redis Error : %v", err.Error()))
		return
	}

	logger.Log().Info(fmt.Sprintf(" On-boarded Servers: %v", onboardedServerIds))

	//calculating/constructing onboarded server...
	onboardedServers := make(map[string]string)
	for _, v := range onboardedServerIds {
		d, ok := serverValues[v]
		if ok {
			onboardedServers[v] = d
		}
	}

	//initialize the SQL server conn...
	connectionListData, mainPageIDs := sqldataprovider.InitConnection(onboardedServers)
	if len(connectionListData) > 0 {
		logger.Log().Info(" SQL server initialized")
	} else {
		logger.Log().Info("SQL server not initialized")
		return
	}

	//assigning to global var
	sqlConnectionListData = connectionListData
	allowedMainPageIDs = mainPageIDs

	//Postgre Connection
	err = postgreprovider.InitPostgreConnection(&model.DBCredentialProvider{
		Host:     helper.GetEnv(helper.PGSHosts),
		Port:     helper.GetEnv(helper.PGSPort),
		UserName: helper.GetEnv(helper.PGSUserName),
		Password: helper.GetEnv(helper.PGSPassword),
		DBName:   helper.GetEnv(helper.PGSDBName),
	})
	if err != nil {
		return
	}

	logger.Log().Info("Initialization Success!!")
}

func assignDefaultValues() {
	redisKeyForRegisteredDevice = helper.GetEnv(helper.RedisKeyForRegisteredDevice)

	redisKeyForTestDevice = helper.GetEnv(helper.RedisKeyForTestDevice)

	redisKeyForCommunicationGroup = helper.GetEnv(helper.RedisKeyForCommunicationGroup)

	singlePartitionDeviceCount = 1000
	singlePartitionDeviceCount, _ = strconv.Atoi(helper.GetEnv(helper.DeviceCountPerPartition))

	redisKeyForDeviceCommandChannel = helper.GetEnv(helper.RedisDeviceCommandChannel)

}

func prepareJob() {
	logger.Log().Info("Job Starting!!")

	//time job interval...
	jobIntervalInSec, err := strconv.ParseInt(helper.GetEnv(helper.JobIntervalInSec), 10, 64)
	if err != nil {
		jobIntervalInSec = 15
	}
	jobIntervalDuration := time.Duration(jobIntervalInSec) * time.Second

	//assigning default values
	assignDefaultValues()

	for {
		if !isAppAlive {
			return
		}

		executeJob()

		//sleeping...
		time.Sleep(jobIntervalDuration)
	}
}

func executeJob() {
	//defining worker..
	workerResponseNotifyChan := make(chan WorkerResponse, len(sqlConnectionListData))
	//workers completed chan
	allCompletedResp := make(chan WorkerPoolResponse)
	//hooking task response collector
	go createResponseCollector(workerResponseNotifyChan, allCompletedResp)

	//generate worker pool, to split and assign tak..
	startWorker(sqlConnectionListData, workerResponseNotifyChan)

	resp := <-allCompletedResp
	//closing chan
	close(allCompletedResp)

	//saving device data ...
	canUpdateDeviceDate := saveDataToStore(resp.registeredDeviceDataList)
	canUpdateSpatialDate := true
	//save spatial data to PostgreGIS
	if len(resp.spatialRequestData.GeofenceData) > 0 || len(resp.spatialRequestData.AreaData) > 0 || len(resp.spatialRequestData.ZoneData) > 0 || len(resp.spatialRequestData.NoGoAreaData) > 0 {

		savedCount, err := postgreprovider.SaveSpatialData(context.Background(), resp.spatialRequestData)
		if err != nil {
			canUpdateSpatialDate = false
			logger.Log().Error(fmt.Sprintf("SaveSpatialData Error : %v", err.Error()))
		}

		logger.Log().Info(fmt.Sprintf("SaveSpatialData Count : %v", savedCount))
	}

	//upd the last-fetch date to DB.....
	// ====================================
	if canUpdateDeviceDate || canUpdateSpatialDate {
		saveJobWorkerStatus(canUpdateDeviceDate, canUpdateSpatialDate, resp.dataFetchRequestData)
	}
}

//close app..
func closeAppSetup() {
	// Wait for ctrl+c
	done := make(chan os.Signal, 1)
	go signal.Notify(done, os.Interrupt)

	<-done

	//stop the app process...
	logger.Log().Info("App trying to stop gracefully!!!")

	isAppAlive = false
	//======== Closing all the connections======....
	logger.Log().Info("Closing DB Conn!!!")

	//closing postgre conn..
	postgreprovider.Close()

	//closing sql server conn..
	for _, d := range sqlConnectionListData {
		sqldataprovider.Close(d.DB)
	}

	//closing redis conn..
	config.Close()

	//closing mongoDB conn..
	entity.Close()

	logger.Log().Info("App stopped successfully!!!")
	os.Exit(0)
}

func startTask(task *model.SQLConnectionData, workerResponseNotifyChan chan<- WorkerResponse, wg *sync.WaitGroup) {

	deviceDataChan := make(chan *model.RegisteredDeviceRequestData)
	spatialDataChan := make(chan *model.SpatialRequestData)

	//getting device data
	go func(c chan<- *model.RegisteredDeviceRequestData) {
		c <- sqldataprovider.GetRegisteredDeviceData(task, allowedMainPageIDs)
	}(deviceDataChan)

	//getting spatial data
	go func(c chan<- *model.SpatialRequestData, serverID string) {
		if serverID != "27" {
			c <- sqldataprovider.GetSpatialData(task)
		} else {
			c <- &model.SpatialRequestData{
				GeofenceData:  make([]*model.GeofenceData, 0),
				AreaData:      make([]*model.AreaData, 0),
				ZoneData:      make([]*model.ZoneData, 0),
				NoGoAreaData:  make([]*model.NoGoAreaData, 0),
				DataFetchDate: time.Now().UTC(),
			}
		}
	}(spatialDataChan, task.ServerID)

	regDeviceData, spatialRequestData := <-deviceDataChan, <-spatialDataChan

	//closing chan...
	close(deviceDataChan)
	close(spatialDataChan)

	//sending response back to chan..
	workerResponseNotifyChan <- WorkerResponse{
		task: task, registeredDeviceRequestData: regDeviceData,
		spatialRequestData: spatialRequestData,
	}

	wg.Done()
}

func startWorker(taskList []*model.SQLConnectionData, workerResponseNotifyChan chan<- WorkerResponse) {
	var wg sync.WaitGroup
	for _, task := range taskList {
		wg.Add(1)
		go startTask(task, workerResponseNotifyChan, &wg)
	}
	wg.Wait()
	//closing chan..
	close(workerResponseNotifyChan)
}

func createResponseCollector(workerResponseNotifyChan <-chan WorkerResponse, done chan<- WorkerPoolResponse) {

	registeredDeviceDataList := make([]*model.RegisteredDeviceData, 0)
	spatialRequestData := &model.SpatialRequestData{
		GeofenceData: make([]*model.GeofenceData, 0),
		AreaData:     make([]*model.AreaData, 0),
		ZoneData:     make([]*model.ZoneData, 0),
		NoGoAreaData: make([]*model.NoGoAreaData, 0),
	}

	dataFetchRequestData := make(map[string]model.DataFetchRequestData)

	for workerresponse := range workerResponseNotifyChan {
		//appending reg.device data
		registeredDeviceDataList = append(registeredDeviceDataList, workerresponse.registeredDeviceRequestData.RegisteredDeviceData...)

		//appending spatial data
		spatialRequestData.GeofenceData = append(spatialRequestData.GeofenceData, workerresponse.spatialRequestData.GeofenceData...)

		spatialRequestData.AreaData = append(spatialRequestData.AreaData, workerresponse.spatialRequestData.AreaData...)

		spatialRequestData.ZoneData = append(spatialRequestData.ZoneData, workerresponse.spatialRequestData.ZoneData...)

		spatialRequestData.NoGoAreaData = append(spatialRequestData.NoGoAreaData, workerresponse.spatialRequestData.NoGoAreaData...)

		//preparing data fetch time, for later update to SQL server
		dataFetchRequestData[workerresponse.task.ServerID] = model.DataFetchRequestData{
			DeviceData:  workerresponse.registeredDeviceRequestData.DataFetchDate,
			SpatialData: workerresponse.spatialRequestData.DataFetchDate,
		}
	}

	//all done, then send back the response to done channel..
	done <- WorkerPoolResponse{
		registeredDeviceDataList: registeredDeviceDataList,
		spatialRequestData:       spatialRequestData,
		dataFetchRequestData:     dataFetchRequestData,
	}
}

//saveDataToStore used to store the data on redis and mongo...
func saveDataToStore(regDeviceData []*model.RegisteredDeviceData) bool {

	canUpdateFetDate := true

	if len(regDeviceData) > 0 {
		getCommunicationGroupFromRedis(regDeviceData, redisKeyForRegisteredDevice)
	}

	registeredData := make(map[string]interface{})
	registeredTestData := make(map[string]interface{})
	removedData := make([]string, 0)
	// removedDataNotifyData := make([]model.DeviceCommmand, 0)
	iotListenerNotifyData := make([]model.DeviceCommmand, 0)
	// devCommGroupData := make([]model.DeviceCommGroupData, 0)

	regMongoDeviceData := make([]model.MongoDeviceData, 0)
	deRegMongoDeviceData := make([]model.MongoDeviceData, 0)

	//already saved communication group details...
	updatedCommGroup := make(map[string]interface{}, 0)
	savedCommunicationGroup, scgerr := config.HGetAll(redisKeyForCommunicationGroup)
	if scgerr != nil {
		logger.Log().Error(fmt.Sprintf("SavedCommunicationGroup [GET] Error : %v", scgerr.Error()))
	}

	for _, data := range regDeviceData {

		//for redis Notification abt changes..
		//we need to intimate only already added device, which would be there in IOT listener local storage, to terminate the connection, if any data changes required!!!!
		if data.CommunicationGroupID >= 0 {

			deviceCommand := model.DeviceCommmand{
				DeviceID:                  data.DeviceID,
				ListenerDeviceCommandType: int32(model.DeviceDataUpdate),
			}

			if data.Active != 1 {
				deviceCommand.ListenerDeviceCommandType = int32(model.DeviceDisconnectionCommand)
			}

			iotListenerNotifyData = append(iotListenerNotifyData, deviceCommand)
		}

		//FOR DIVERSION COMM-GROUP CALC .......
		//calculate Comm-Group for diversion data...
		if len(data.DiversionDetails) > 0 {

			//bulk comm. group calculating...
			diversionDetails, savedCGroup, updCGroup := calculateCommunicationGroupForDiversionData(data.DiversionDetails, savedCommunicationGroup, updatedCommGroup, singlePartitionDeviceCount, (data.Active != 1))

			data.DiversionDetails = diversionDetails
			savedCommunicationGroup = savedCGroup
			updatedCommGroup = updCGroup

		}

		//ACTUAL Device processing ....
		if data.Active == 1 {
			if data.CommunicationGroupID < 0 {
				//chk commun.groupuno already generated or not..
				communicationGroup := 0
				//calculating comm. group
				communicationGroup, savedCommunicationGroup = calculateCommunicationGroup(savedCommunicationGroup, singlePartitionDeviceCount)
				//add/upd. list to find out modified list..
				updatedCommGroup[strconv.Itoa(communicationGroup)] = ""

				//assign calculated comm.group value
				data.CommunicationGroupID = communicationGroup

			}

			jsonData, err := json.Marshal(data)
			if err != nil {
				logger.Log().Error(fmt.Sprintf("startDeviceDataTransferJob Error : %v", err.Error()))
				//start next iteration
				continue
			}

			//all active devices present in reg.dev.data redis key => active + test
			registeredData[data.DeviceID] = string(jsonData)

			//only add installed active devices into mongodb...
			if data.DeviceMasterStatusUno == 5 {
				//mongo active dev...
				regMongoDeviceData = append(regMongoDeviceData, model.MongoDeviceData{
					ID:                   data.DeviceID,
					ProviderTenantUIDs:   data.ProviderTenantUIDs,
					TenantGroupUID:       data.TenantGroupUID,
					TenantUID:            data.TenantUID,
					TenantName:           data.TenantName,
					DeviceTypeID:         data.DeviceTypeID,
					CommunicationGroupID: data.CommunicationGroupID,
					VehicleID:            data.VehicleID,
					DiversionDetails:     data.DiversionDetails,
				})

			} else {
				//store devices which is not yet installed on vehicle added under testdevicedata redis key..
				registeredTestData[data.DeviceID] = string(jsonData)

			}

		} else {
			//de-allocate communication group....
			currentCommunicationGroup := strconv.Itoa(data.CommunicationGroupID)
			currentCommunicationGroupID, err := strconv.Atoi(currentCommunicationGroup)
			if err != nil {
				currentCommunicationGroupID = -1
				logger.Log().Error(fmt.Sprintf("saveDataToStore (strconv.Atoi) Error : %v", err.Error()))
			}
			//if comm. group is present then proceed...
			if currentCommunicationGroupID >= 0 {
				//getting curr. val
				currentCount, _ := strconv.Atoi(savedCommunicationGroup[currentCommunicationGroup])
				//setting new val(by removing)
				savedCommunicationGroup[currentCommunicationGroup] = strconv.Itoa(currentCount - 1)
				//add/upd. list to find out modified list..
				updatedCommGroup[currentCommunicationGroup] = ""

				removedData = append(removedData, data.DeviceID)

				// /mongo
				deRegMongoDeviceData = append(deRegMongoDeviceData, model.MongoDeviceData{
					ID: data.DeviceID,
				})
			}
		}
	}

	//saving all the device data into reg.dev.data redis key[main one]
	if len(registeredData) > 0 {
		redisErr := config.HMSet(redisKeyForRegisteredDevice, registeredData)
		if redisErr != nil {
			logger.Log().Error(fmt.Sprintf("startDevicveDataTransferJob (Redis Save) Error : %v", redisErr.Error()))
		}

		logger.Log().Info(fmt.Sprintf("SAVED DEVICE COUNT : %v", len(registeredData)))
	}

	//mongo DB saving
	if len(regMongoDeviceData) > 0 || len(deRegMongoDeviceData) > 0 {
		rowCount, mErr := entity.BulkWrite("tblvehiclerecentupdates", regMongoDeviceData, deRegMongoDeviceData)
		if mErr != nil {
			logger.Log().Error(fmt.Sprintf("startDevicveDataTransferJob (Mongo Bulk) Error : %v", mErr.Error()))
		}

		if rowCount > 0 {
			logger.Log().Info(fmt.Sprintf("MONGO COUNT: %v", rowCount))
		}

		// deleting test device data
		if len(regMongoDeviceData) > 0 {
			removedTestData := make([]string, 0)

			for _, r := range regMongoDeviceData {
				removedTestData = append(removedTestData, r.ID)
			}

			// removedData = append(removedData, data.DeviceID)
			if len(removedTestData) > 0 {
				count, redisErr1 := config.HDel(redisKeyForTestDevice, removedTestData)
				if redisErr1 != nil {
					logger.Log().Error(fmt.Sprintf("startDevicveDataTransferJob (TEST DEVICE(Mongo) DELETE) Error : %v", redisErr1.Error()))
				}

				if count > 0 {
					logger.Log().Info(fmt.Sprintf("DELETED TEST DEVICE(Mongo) COUNT : %v", len(removedTestData)))
				}
			}
		}
	}

	//saving test device data
	if len(registeredTestData) > 0 {
		redisErr1 := config.HMSet(redisKeyForTestDevice, registeredTestData)
		if redisErr1 != nil {
			logger.Log().Error(fmt.Sprintf("startDeviveDataTransferJob (Redis Test Device Save) Error : %v", redisErr1.Error()))
		}

		logger.Log().Info(fmt.Sprintf("SAVED TEST DEVICE COUNT : %v", len(registeredTestData)))
	}

	// deleting data
	if len(removedData) > 0 {
		count, redisErr := config.HDel(redisKeyForRegisteredDevice, removedData)
		if redisErr != nil {
			logger.Log().Error(fmt.Sprintf("startDevicveDataTransferJob (DELETE) Error : %v", redisErr.Error()))
		} else if count < 1 {
			logger.Log().Warn("no device deleted!")
		}

		logger.Log().Info(fmt.Sprintf("DELETED DEVICE COUNT : %v", len(removedData)))

		// deleting test device data
		if len(removedData) > 0 {
			count, redisErr1 := config.HDel(redisKeyForTestDevice, removedData)
			if redisErr1 != nil {
				logger.Log().Error(fmt.Sprintf("startDevicveDataTransferJob (TEST DEVICE DELETE) Error : %v", redisErr1.Error()))
			}

			if count > 0 {
				logger.Log().Info(fmt.Sprintf("DELETED TEST DEVICE COUNT : %v", len(removedData)))
			}
		}


	}

	//2020-JUL-13 (Fredin)
	//notifying IOT Listener about the changes...
	if len(iotListenerNotifyData) > 0 {
		byteRes, ee := json.Marshal(iotListenerNotifyData)
		if ee != nil {
			logger.Log().Error(fmt.Sprintf("IOT_LISTENER_NOTIFY_DATA (Marshal Redis) Error : %v", ee.Error()))
		} else {
			//send data
			kRes, kErr := config.Publish(redisKeyForDeviceCommandChannel, byteRes)
			if kErr != nil {
				logger.Log().Error(fmt.Sprintf("IOT_LISTENER_NOTIFY_DATA (Redis PUB) Error : %v", kErr.Error()))
			}

			if kRes > 0 {
				logger.Log().Info(fmt.Sprintf("IOT_LISTENER_NOTIFY_DATA DEVICE PUSHED COUNT: %v", len(iotListenerNotifyData)))

				for _, a := range iotListenerNotifyData {
					logger.Log().Info(fmt.Sprintf("DeviceID: %v ===> CT: %v", a.DeviceID, a.ListenerDeviceCommandType))
				}

			}
		}
	}

	//update communication group details back to redis...
	if len(updatedCommGroup) > 0 {
		for k := range updatedCommGroup {
			updatedCommGroup[k] = savedCommunicationGroup[k]
		}

		scgerr = config.HMSet(redisKeyForCommunicationGroup, updatedCommGroup)
		if scgerr != nil {
			canUpdateFetDate = false

			logger.Log().Error(fmt.Sprintf("SavedCommunicationGroup [FINAL SAVE] Error : %v", scgerr.Error()))
		}

		logger.Log().Info(fmt.Sprintf("Comm.group updated successfully [REDIS] count : %v", len(updatedCommGroup)))
	}

	return canUpdateFetDate
}

//calculating comm. group
func calculateCommunicationGroupForDiversionData(diversionData []*model.DeviceDiversionData, savedCommunicationGroup map[string]string, updatedCommGroup map[string]interface{}, singlePartitionDeviceCount int, isDeleteCall bool) ([]*model.DeviceDiversionData, map[string]string, map[string]interface{}) {

	resp := make([]*model.DeviceDiversionData, 0)

	for _, data := range diversionData {

		//Active Records
		if data.Active == 1 && !isDeleteCall {

			if data.CommunicationGroupID < 0 {
				//chk commun.groupuno already generated or not..
				communicationGroup := 0
				//calculating comm. group
				communicationGroup, savedCommunicationGroup = calculateCommunicationGroup(savedCommunicationGroup, singlePartitionDeviceCount)
				//add/upd. list to find out modified list..
				updatedCommGroup[strconv.Itoa(communicationGroup)] = ""

				//assign calculated comm.group value
				data.CommunicationGroupID = communicationGroup
			}
			//adding active data..
			resp = append(resp, data)

		} else {
			//de-allocate communication group....
			currentCommunicationGroup := strconv.Itoa(data.CommunicationGroupID)
			currentCommunicationGroupID, err := strconv.Atoi(currentCommunicationGroup)
			if err != nil {
				currentCommunicationGroupID = -1
				logger.Log().Error(fmt.Sprintf("calculateCommunicationGroupForDiversionData (strconv.Atoi) Error : %v", err.Error()))
			}

			//if comm. group is present then proceed...
			if currentCommunicationGroupID >= 0 {
				//getting curr. val
				currentCount, _ := strconv.Atoi(savedCommunicationGroup[currentCommunicationGroup])
				//setting new val(by removing)
				savedCommunicationGroup[currentCommunicationGroup] = strconv.Itoa(currentCount - 1)
				//add/upd. list to find out modified list..
				updatedCommGroup[currentCommunicationGroup] = ""
			}
		}
	}

	return resp, savedCommunicationGroup, updatedCommGroup
}

//calculating comm. group
func calculateCommunicationGroup(masterList map[string]string, singlePartitionDeviceCount int) (int, map[string]string) {

	for k, v := range masterList {
		cGroupValCount, _ := strconv.Atoi(v)
		if cGroupValCount < singlePartitionDeviceCount {
			if k == "" { //in-case
				k = "0"
			}
			cGroup, _ := strconv.Atoi(k)
			masterList[k] = strconv.Itoa(cGroupValCount + 1)
			return cGroup, masterList
		}
	}

	//if not found add new comm.group val....
	//adding entry in masterlist
	k := 0
	//calc next possi comm.group
	for idex := range make([]int, len(masterList)) {
		_, ok := masterList[strconv.Itoa(idex)]
		if !ok {
			k = idex
			break
		}
	}

	if len(masterList) > 0 && k == 0 {
		k = len(masterList)
	}
	masterList[strconv.Itoa(k)] = "1"

	return k, masterList
}

func filter(regData []model.RegisteredDeviceData, deviceID string) (model.RegisteredDeviceData, bool) {
	var empty model.RegisteredDeviceData
	for _, v := range regData {
		if v.DeviceID == deviceID {
			return v, true
		}
	}
	return empty, false
}

func filterDiversionData(dd []*model.DeviceDiversionData, tenantGroupUID string, tenantUID string) (*model.DeviceDiversionData, bool) {
	var empty *model.DeviceDiversionData
	for _, v := range dd {
		if v.TenantGroupUID == tenantGroupUID && v.TenantUID == tenantUID {
			return v, true
		}
	}
	return empty, false
}

func getCommunicationGroupFromRedis(regData []*model.RegisteredDeviceData, redisKeyForRegisteredDevice string) {
	inputData := make([]string, 0)
	redisStoredData := make([]model.RegisteredDeviceData, 0)

	for _, row := range regData {
		inputData = append(inputData, row.DeviceID)
	}

	//finding respective communication group from redis...
	resp, err := config.HMGet(redisKeyForRegisteredDevice, inputData)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("getCommunicationGroupFromRedis [Redis] Error : %v", err.Error()))
	}

	//de-serializing data...
	for _, v := range resp {
		if v != nil {
			var output model.RegisteredDeviceData
			err = json.Unmarshal([]byte(v.(string)), &output)
			if err != nil {
				logger.Log().Error(fmt.Sprintf("getCommunicationGroupFromRedis unmarshal : %v", err.Error()))
			} else {
				redisStoredData = append(redisStoredData, output)
			}
		}
	}

	if len(redisStoredData) > 0 {
		for _, row := range regData {
			data, ok := filter(redisStoredData, row.DeviceID)
			if ok {
				row.CommunicationGroupID = data.CommunicationGroupID
			} else {
				row.CommunicationGroupID = -1
			}

			//for diversion data comm.group logic...
			if len(row.DiversionDetails) > 0 || len(data.DiversionDetails) > 0 {

				for rIndex, childRow := range row.DiversionDetails {
					//default values...
					row.DiversionDetails[rIndex].Active = 1
					row.DiversionDetails[rIndex].CommunicationGroupID = -1

					childData, hasData := filterDiversionData(data.DiversionDetails, childRow.TenantGroupUID, childRow.TenantUID)

					if hasData {
						row.DiversionDetails[rIndex].CommunicationGroupID = childData.CommunicationGroupID
					}
				}

				//checking for inactive (removed diversion config)
				if len(row.DiversionDetails) != len(data.DiversionDetails) {
					for _, childRow := range data.DiversionDetails {

						_, hasData := filterDiversionData(row.DiversionDetails, childRow.TenantGroupUID, childRow.TenantUID)

						if !hasData {
							childRow.Active = 0
							row.DiversionDetails = append(row.DiversionDetails, childRow)
						}
					}
				}
			}
		}
	}

}

//saveJobWorkerStatus saving/updating back to Sql Server abt last fetch date....
func saveJobWorkerStatus(canUpdateDeviceDate bool, canUpdateSpatialDate bool, dataFetchRequestData map[string]model.DataFetchRequestData) {

	errOccured := false
	var wg sync.WaitGroup
	for _, task := range sqlConnectionListData {

		v, ok := dataFetchRequestData[task.ServerID]
		if ok {
			//adding waitgroup...
			wg.Add(1)

			go func(updDeviceDate bool, updSpatialDate bool, dataFetchData model.DataFetchRequestData, t *model.SQLConnectionData, resolver *sync.WaitGroup) {

				_, err := sqldataprovider.UpdateDataSyncFetchDate(t, updDeviceDate, dataFetchData.DeviceData, updSpatialDate, dataFetchData.SpatialData)
				//error occured
				if err != nil {
					errOccured = true
				}
				//resolving...
				resolver.Done()
			}(canUpdateDeviceDate, canUpdateSpatialDate, v, task, &wg)
		}

	}
	//awaiting for all to complete...
	wg.Wait()

	if errOccured {
		updSysncDateErrorCount = updSysncDateErrorCount + 1
	}

	//stop app...
	if updSysncDateErrorCount > 10 {
		logger.Log().Panic("updSysncDateErrorCount -> exceeded the max limit!!!")
	}
}
