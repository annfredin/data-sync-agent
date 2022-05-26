package sqldataprovider

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"data-sync-agent/crypto"
	model "data-sync-agent/model"
	"data-sync-agent/utils/logger"

	mssql "github.com/denisenkom/go-mssqldb"
)

//InitConnection ...
func InitConnection(values map[string]string) ([]*model.SQLConnectionData, string) {

	sqlConnectionList := make([]*model.SQLConnectionData, 0)
	appMainPageID := make([]string, 0)

	for key, value := range values {
		//deserializing data
		sqlCredentialProvider := &model.SQLCredentialProvider{}
		connStringHexData, err := crypto.Decrypt(value, crypto.W1EncType)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("InitConnection Deserialization[connStringHexData] Error : %v", err.Error()))
			continue
		}
		connStringData, err := hex.DecodeString(connStringHexData)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("InitConnection Deserialization[connStringData] Error : %v", err.Error()))
			continue
		}

		err = json.Unmarshal(connStringData, sqlCredentialProvider)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("InitConnection Deserialization Error : %v", err.Error()))
			continue
		}

		db, err := getSQLServerCon(sqlCredentialProvider)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("InitConnection getSQLServerCon Error : %v", err.Error()))
			continue
		}

		//adding list value
		sqlConnectionList = append(sqlConnectionList, &model.SQLConnectionData{
			ServerID: key,
			DB:       db,
		})

		appMainPageID = append(appMainPageID, sqlCredentialProvider.MainPageID)
	}

	//return status
	return sqlConnectionList, strings.Join(appMainPageID, ",")
}

//getSQLServerCon ...
func getSQLServerCon(sqlCredentialProvider *model.SQLCredentialProvider) (*sql.DB, error) {
	query := url.Values{}
	query.Add("database", sqlCredentialProvider.DBName)

	u := &url.URL{
		Scheme:   "sqlserver",
		Host:     sqlCredentialProvider.HostName,
		User:     url.UserPassword(sqlCredentialProvider.UserName, sqlCredentialProvider.Password),
		RawQuery: query.Encode(),
	}

	// Create a new connector object by calling NewConnector
	connector, err := mssql.NewConnector(u.String())
	if err != nil {
		return nil, err
	}

	//open SQL server connection
	return sql.OpenDB(connector), nil
}

func getValue(pval *interface{}) interface{} {
	switch v := (*pval).(type) {
	default:
		return v
	}
}

func getDiversionData(pval *interface{}) []*model.DeviceDiversionData {
	respDiversionData := make([]*model.DeviceDiversionData, 0)
	strRequest := (*pval).(string)

	if len(strings.TrimSpace(strRequest)) == 0 {
		return respDiversionData
	}

	if err := json.Unmarshal([]byte(strRequest), &respDiversionData); err != nil {
		logger.Log().Error(fmt.Sprintf("getDiversionData:Parsing, Error : %v", err.Error()))

		return respDiversionData
	}

	return respDiversionData

}

//GetRegisteredDeviceData ...
func GetRegisteredDeviceData(connData *model.SQLConnectionData, allowedMainPageIDs string) *model.RegisteredDeviceRequestData {

	dataFetchedOn := time.Now().UTC()
	hasRows := false

	regDeviceData := make([]*model.RegisteredDeviceData, 0)

	registeredDeviceRequestData := &model.RegisteredDeviceRequestData{
		RegisteredDeviceData: make([]*model.RegisteredDeviceData, 0),
	}

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := connData.DB.QueryContext(ctx, "GetIOTRegisteredDeviceData", sql.Named("ApplicationMainPageID", allowedMainPageIDs), sql.Named("DataFetchedOn", sql.Out{Dest: &dataFetchedOn}))
	if err != nil {
		logger.Log().Error(fmt.Sprintf("GetRegisteredDeviceData, Server=%v Error : %v", connData.ServerID, err.Error()))
		return registeredDeviceRequestData
	}

	//assigning dataFetchedOn
	registeredDeviceRequestData.DataFetchDate = dataFetchedOn
	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err = rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("GetRegisteredDeviceData Server=%v  rows.Next() Error : %v", connData.ServerID, err.Error()))
			return registeredDeviceRequestData
		}
		for i, c := range resultValue {
			if columns[i] != "diversiondetails" {
				convertedRow[columns[i]] = getValue(c.(*interface{}))
			} else {
				convertedRow[columns[i]] = getDiversionData(c.(*interface{}))
			}
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return registeredDeviceRequestData
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("GetRegisteredDeviceData Server=%v Marshal Error : %v", connData.ServerID, err.Error()))
		return registeredDeviceRequestData
	}

	if err := json.Unmarshal(jsonbody, &regDeviceData); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("GetRegisteredDeviceData Server=%v UnMarshal Error : %v", connData.ServerID, err.Error()))
	}

	//assigning reg device data...
	registeredDeviceRequestData.RegisteredDeviceData = regDeviceData

	return registeredDeviceRequestData
}

//UpdRegisteredDeviceData ...
func UpdRegisteredDeviceData(db *sql.DB, deviceCommGroupData []model.DeviceCommGroupData) (bool, error) {

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outParameter := 0
	outErrorMessage := ""

	tvpDeviceData := mssql.TVP{
		TypeName: "TypIntString",
		Value:    deviceCommGroupData,
	}

	_, err := db.ExecContext(ctx, "SpUpdRegisteredDeviceData",
		sql.Named("TvpRegDeviceData", tvpDeviceData),
		sql.Named("OutParameter", sql.Out{Dest: &outParameter}),
		sql.Named("OutErrorMessage", sql.Out{Dest: &outErrorMessage}))

	if err != nil {
		return false, err
	}

	return outParameter > 0, errors.New(outErrorMessage)
}

//Getiottest ...
func Getiottest(db *sql.DB) []map[string]string {

	regDeviceData := make([]map[string]string, 0)

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := db.QueryContext(ctx, "iottest")
	if err != nil {
		logger.Log().Error(fmt.Sprintf("iotest Error : %v", err.Error()))
		return regDeviceData
	}

	for rows.Next() {
		var res, res1 string
		err = rows.Scan(&res, &res1)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("iottest rows.Next() Error : %v", err.Error()))
			return regDeviceData
		}
		regDeviceData = append(regDeviceData, map[string]string{
			res: res1,
		})
	}

	return regDeviceData
}

//Getiottest1 ...
func Getiottest1(db *sql.DB) []string {

	regDeviceData := make([]string, 0)

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := db.QueryContext(ctx, "iottest1")
	if err != nil {
		logger.Log().Error(fmt.Sprintf("iotest Error : %v", err.Error()))
		return regDeviceData
	}

	for rows.Next() {
		var resstring string
		err = rows.Scan(&resstring)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("iottest rows.Next() Error : %v", err.Error()))
			return regDeviceData
		}
		regDeviceData = append(regDeviceData, resstring)
	}

	return regDeviceData
}

//GetSpatialData ...
func GetSpatialData(connData *model.SQLConnectionData) *model.SpatialRequestData {
	dataFetchedOn := time.Now().UTC()

	spatialRequestData := &model.SpatialRequestData{
		GeofenceData: make([]*model.GeofenceData, 0),
		AreaData:     make([]*model.AreaData, 0),
		ZoneData:     make([]*model.ZoneData, 0),
		NoGoAreaData: make([]*model.NoGoAreaData, 0),
	}

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rows, err := connData.DB.QueryContext(ctx, "GetIOTSpatialData", sql.Named("DataFetchedOn", sql.Out{Dest: &dataFetchedOn}))
	if err != nil {
		logger.Log().Error(fmt.Sprintf("GetSpatialData Server=%v Error : %v", connData.ServerID, err.Error()))
		return spatialRequestData
	}
	spatialRequestData.DataFetchDate = dataFetchedOn
	spatialRequestData.GeofenceData = parseGeofenceData(rows)

	//checking has next result, and iterating based oon status
	if rows.NextResultSet() {
		spatialRequestData.AreaData = parseAreaData(rows)
	}
	if rows.NextResultSet() {
		spatialRequestData.ZoneData = parseZoneData(rows)
	}
	if rows.NextResultSet() {
		spatialRequestData.NoGoAreaData = parseNoGoAreaData(rows)
	}

	//returning result-set
	return spatialRequestData
}

//parse geofence data ....
func parseGeofenceData(rows *sql.Rows) []*model.GeofenceData {
	hasRows := false
	geofenceData := make([]*model.GeofenceData, 0)

	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err := rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("parseGeofenceData rows.Next() Error : %v", err.Error()))
			return geofenceData
		}
		for i, c := range resultValue {
			convertedRow[columns[i]] = getValue(c.(*interface{}))
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return geofenceData
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("parseGeofenceData Marshal Error : %v", err.Error()))
		return geofenceData
	}

	if err := json.Unmarshal(jsonbody, &geofenceData); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("parseGeofenceData UnMarshal Error : %v", err.Error()))
	}

	return geofenceData
}

func parseAreaData(rows *sql.Rows) []*model.AreaData {
	hasRows := false
	areaData := make([]*model.AreaData, 0)

	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err := rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("parseAreaData rows.Next() Error : %v", err.Error()))
			return areaData
		}
		for i, c := range resultValue {
			convertedRow[columns[i]] = getValue(c.(*interface{}))
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return areaData
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("parseAreaData Marshal Error : %v", err.Error()))
		return areaData
	}

	if err := json.Unmarshal(jsonbody, &areaData); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("parseAreaData UnMarshal Error : %v", err.Error()))
	}

	return areaData
}

func parseZoneData(rows *sql.Rows) []*model.ZoneData {
	hasRows := false
	zoneData := make([]*model.ZoneData, 0)

	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err := rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("parseZoneData rows.Next() Error : %v", err.Error()))
			return zoneData
		}
		for i, c := range resultValue {
			convertedRow[columns[i]] = getValue(c.(*interface{}))
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return zoneData
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("parseZoneData Marshal Error : %v", err.Error()))
		return zoneData
	}

	if err := json.Unmarshal(jsonbody, &zoneData); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("parseZoneData UnMarshal Error : %v", err.Error()))
	}

	return zoneData
}

func parseNoGoAreaData(rows *sql.Rows) []*model.NoGoAreaData {
	hasRows := false
	noGoAreaData := make([]*model.NoGoAreaData, 0)

	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err := rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("parseNoGoAreaData rows.Next() Error : %v", err.Error()))
			return noGoAreaData
		}
		for i, c := range resultValue {
			convertedRow[columns[i]] = getValue(c.(*interface{}))
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return noGoAreaData
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("parseNoGoAreaData Marshal Error : %v", err.Error()))
		return noGoAreaData
	}

	if err := json.Unmarshal(jsonbody, &noGoAreaData); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("parseNoGoAreaData UnMarshal Error : %v", err.Error()))
	}

	return noGoAreaData
}

//UpdateDataSyncFetchDate ...
func UpdateDataSyncFetchDate(connData *model.SQLConnectionData, canUpdateDeviceDate bool,
	deviceDate time.Time,
	canUpdateSpatialDate bool,
	spatialDate time.Time) (bool, error) {
	//upd status(out var)
	updStatus := false
	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := connData.DB.ExecContext(ctx, "UpdIOTDataSyncFetchDate",
		sql.Named("CanUpdateDeviceDate", canUpdateDeviceDate),
		sql.Named("DeviceDate", deviceDate),
		sql.Named("CanUpdateSpatialDate", canUpdateSpatialDate),
		sql.Named("SpatialDate", spatialDate),
		sql.Named("Status", sql.Out{Dest: &updStatus}))
	if err != nil {
		logger.Log().Error(fmt.Sprintf("UpdateDataSyncFetchDate Server=%v , DeviceDate=%v, SpatialDate=%v, Error : %v", connData.ServerID, deviceDate, spatialDate, err.Error()))
		return false, err
	}

	return updStatus, nil
}

//GET UNAUTH DATA ...
//parseUnAuthDeviceData ...
func parseUnAuthDeviceData(rows *sql.Rows) []model.UnAuthDeviceResponse {
	hasRows := false
	unAuthDeviceResponse := make([]model.UnAuthDeviceResponse, 0)

	//columns processing
	columns, _ := rows.Columns()
	resultValue := make([]interface{}, len(columns))
	for i := range columns {
		resultValue[i] = new(interface{})
	}

	//data processing...
	resultMap := make([]map[string]interface{}, 0)
	for rows.Next() {
		hasRows = true
		convertedRow := make(map[string]interface{}, 0)
		err := rows.Scan(resultValue...)
		if err != nil {
			logger.Log().Error(fmt.Sprintf("parseUnAuthDeviceData rows.Next() Error : %v", err.Error()))
			return unAuthDeviceResponse
		}
		for i, c := range resultValue {
			convertedRow[columns[i]] = getValue(c.(*interface{}))
		}
		resultMap = append(resultMap, convertedRow)
	}

	//if no record to process
	if !hasRows {
		return unAuthDeviceResponse
	}

	jsonbody, err := json.Marshal(resultMap)
	if err != nil {
		logger.Log().Error(fmt.Sprintf("parseUnAuthDeviceData Marshal Error : %v", err.Error()))
		return unAuthDeviceResponse
	}

	if err := json.Unmarshal(jsonbody, &unAuthDeviceResponse); err != nil {
		// do error check
		logger.Log().Error(fmt.Sprintf("parseUnAuthDeviceData UnMarshal Error : %v", err.Error()))
	}

	return unAuthDeviceResponse
}

//GET UNAUTH DATA ...
func GetUnAuthDeviceDetails(connData *model.SQLConnectionData, deviceList []string) []model.UnAuthDeviceResponse {

	unAuthDeviceResponse := make([]model.UnAuthDeviceResponse, 0)

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	typIntStringVal := make([]model.TVPIntStr, 0)
	for _, v := range deviceList {
		typIntStringVal = append(typIntStringVal, model.TVPIntStr{
			DataUno:   0,
			DataValue: v,
		})
	}

	tvpType := mssql.TVP{
		TypeName: "TypIntString",
		Value:    typIntStringVal,
	}
	rows, err := connData.DB.QueryContext(ctx, "GetIOTUnAuthDeviceDetails", sql.Named("DeviceList", tvpType))
	if err != nil {
		logger.Log().Error(fmt.Sprintf("GetUnAuthDeviceDetails Server=%v Error : %v", connData.ServerID, err.Error()))
		return unAuthDeviceResponse
	}

	unAuthDeviceResponse = parseUnAuthDeviceData(rows)

	//returning result-set
	return unAuthDeviceResponse
}

func UpdUnAuthDeviceDetails(connData *model.SQLConnectionData, applicationServerID int, deviceList []model.UnAuthDeviceResponse) int64 {

	//creating context for trans...
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tvpType := mssql.TVP{
		TypeName: "TYP_UNAUTH_DATA",
		Value:    deviceList,
	}
	_, err := connData.DB.QueryContext(ctx, "UpdIOTUnAuthDeviceDetails", sql.Named("Inserted_ApplicationServerID", applicationServerID), sql.Named("DeviceList", tvpType))
	if err != nil {
		logger.Log().Error(fmt.Sprintf("GetUnAuthDeviceDetails Server=%v Error : %v", connData.ServerID, err.Error()))
		return 0
	}

	//returning result-set
	return 1
}

//Close ...
func Close(db *sql.DB) {
	db.Close()
}
