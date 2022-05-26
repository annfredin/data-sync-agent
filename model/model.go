package model

import (
	"database/sql"
	"time"
)

//RegisteredDeviceRequestData ...
type RegisteredDeviceRequestData struct {
	RegisteredDeviceData []*RegisteredDeviceData
	DataFetchDate        time.Time
}

//DeviceDiversionData ...
type DeviceDiversionData struct {
	TenantGroupUID       string `json:"tenantgroupuid"`
	TenantUID            string `json:"tenantuid"`
	CommunicationGroupID int    `json:"communicationgroupid"`
	ProviderTenantUIDs   string `json:"providertenantuids"`
	Active               int    `json:"active"`
}

//RegisteredDeviceData ...
type RegisteredDeviceData struct {
	DeviceID              string `json:"deviceid"`
	VehicleID             string `json:"vehicleid"`
	CommunicationGroupID  int    `json:"communicationgroupid"`
	BatchProcessGroupID   int    `json:"batchprocessgroupid"`
	TenantGroupUID        string `json:"tenantgroupuid"`
	TenantUID             string `json:"tenantuid"`
	ProviderTenantUIDs    string `json:"providertenantuids"`
	ParserID              int    `json:"parserid"`
	DeviceMasterStatusUno int    `json:"devicemasterstatusuno"`
	DeviceTypeID          int    `json:"devicetypeid"`
	TenantName            string `json:"tenantname"`
	Active                int    `json:"active"`

	DiversionDetails []*DeviceDiversionData `json:"diversiondetails"`
}

//DeviceCommGroupData ...
type DeviceCommGroupData struct {
	DataUno   int
	DataValue string
}

//DBCredentialProvider ...
type DBCredentialProvider struct {
	Host     string
	Port     string
	DBName   string
	UserName string
	Password string
}

//SQLCredentialProvider ...
type SQLCredentialProvider struct {
	HostName   string `json:"hostname"`
	DBName     string `json:"dbname"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
	MainPageID string `json:"mainpageid"`
}

//SQLConnectionData ...
type SQLConnectionData struct {
	ServerID string
	DB       *sql.DB
}

//ListenerDeviceCommandType command type received in the consumer
type ListenerDeviceCommandType int

const (
	//DeviceCommand the command which should be pushed to the device
	DeviceCommand ListenerDeviceCommandType = 1 + iota
	//DeviceDisconnectionCommand disconnect the device if connected
	DeviceDisconnectionCommand
	//DeviceDataUpdate ...
	DeviceDataUpdate
)

//DeviceCommmand struct which is received in listener to push to device
type DeviceCommmand struct {
	DeviceID                  string `json:"deviceid"`
	ListenerDeviceCommandType int32  `json:"listenerdevicecommandtype"`
}

//MongoDeviceData ...
type MongoDeviceData struct {
	ID                   string                 `bson:"_id,omitempty"`
	TenantUID            string                 `json:"tenantuid"`
	TenantGroupUID       string                 `json:"tenantgroupuid"`
	ProviderTenantUIDs   string                 `json:"providertenantuids"`
	DeviceTypeID         int                    `json:"devicetypeid"`
	CommunicationGroupID int                    `json:"communicationgroupid"`
	TenantName           string                 `json:"tenantname"`
	VehicleID            string                 `json:"vehicleid"`
	DiversionDetails     []*DeviceDiversionData `json:"diversiondetails"`
}

//Postgre == Geo spatial models....

//SpatialRequestData ...
type SpatialRequestData struct {
	GeofenceData  []*GeofenceData
	AreaData      []*AreaData
	ZoneData      []*ZoneData
	NoGoAreaData  []*NoGoAreaData
	DataFetchDate time.Time
}

//GeofenceData ...
type GeofenceData struct {
	TenantUID        string
	TenantGroupUID   string
	GeofenceUID      string
	GeofenceNameEN   string
	GeofenceNameOL   string
	GeofenceTypeID   int
	IsApproved       int
	Active           int
	LastModifiedDate time.Time
	OgrGeometry      string
}

//AreaData ...
type AreaData struct {
	TenantUID        string
	TenantGroupUID   string
	AreaUID          string
	Active           int
	LastModifiedDate time.Time
	OgrGeometry      string
}

//ZoneData ...
type ZoneData struct {
	TenantUID        string
	TenantGroupUID   string
	ZoneUID          string
	Active           int
	LastModifiedDate time.Time
	OgrGeometry      string
}

//NoGoAreaData  ...
type NoGoAreaData struct {
	TenantUID           string
	TenantGroupUID      string
	NoGoAreaGeofenceUID string
	Active              int
	LastModifiedDate    time.Time
	OgrGeometry         string
}

//DataFetchRequestData ...
type DataFetchRequestData struct {
	DeviceData  time.Time
	SpatialData time.Time
}

//UnAuthDeviceResponse ...
type UnAuthDeviceResponse struct {
	DeviceID            string
	ApplicationServerID int
	Active              int
	DeviceCount         int
	TrnStatus           int
	DeviceTypeName      string
	CompanyName         string
	VehicleID           string
	IsDiverted          int
}

//TVPIntStr ...
type TVPIntStr struct {
	DataUno   int
	DataValue string
}
