package postgreprovider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"data-sync-agent/helper"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"

	model "data-sync-agent/model"
	"data-sync-agent/utils/logger"
)

var dbc *pgx.Conn

//InitPostgreConnection ...
func InitPostgreConnection(cp *model.DBCredentialProvider) error {

	//preparing conn settings...
	port, _ := strconv.ParseUint(cp.Port, 10, 16)
	config, _ := pgx.ParseConfig("")
	config.Host = cp.Host
	config.Port = uint16(port)
	config.User = cp.UserName
	config.Password = cp.Password
	config.Database = cp.DBName
	config.TLSConfig = nil
	config.Fallbacks = make([]*pgconn.FallbackConfig, 0)
	config.LookupFunc = func(ctx context.Context, host string) (addrs []string, err error) {
		return strings.Split(host, ","), nil
	}


	if nOK := helper.HasEmpty(cp.Port,cp.DBName,cp.Host,cp.UserName,cp.Password); nOK {
		logger.Log().Fatal("PostgreSQL Connection Secret Error")
		return nil
	}

	conn, err := pgx.ConnectConfig(context.Background(), config)
	if err != nil {
		logger.Log().Fatal(fmt.Sprintf("InitPostgreConnection  Error : %v", err.Error()))

		return err
	}
	err = conn.Ping(context.Background())
	if err != nil {
		logger.Log().Fatal(fmt.Sprintf("InitPostgreConnection  Ping Error : %v", err.Error()))
		return err

	}

	//assignning obj..
	dbc = conn
	logger.Log().Info("Postgre Connection Successful!!!")

	return nil
}

//SaveSpatialData ...
func SaveSpatialData(ctx context.Context, spatialDataRequestData *model.SpatialRequestData) (int64, error) {

	recordAffected := int64(0)

	currentTime := time.Now().UTC()
	//creating trans
	tx, _ := dbc.Begin(ctx)
	batch := &pgx.Batch{}

	//constructing geofence data...
	for _, data := range spatialDataRequestData.GeofenceData {

		if len(strings.TrimSpace(data.OgrGeometry)) > 0 {
			batch.Queue(`INSERT INTO tblmstgeofence (tenantuid,tenantgroupuid,geofenceuid,geofencenameen, geofencenameol,geofencetypeid,isapproved,active,lastmodifieddate, ogr_geometry, ogr_geography, createdon)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, ST_GeomFromText($10,4326), ST_GeographyFromText($10),$11) ON CONFLICT ON CONSTRAINT tblmstgeofence_pkey DO UPDATE SET geofencenameen = excluded.geofencenameen, geofencenameol = excluded.geofencenameol,geofencetypeid = excluded.geofencetypeid,isapproved = excluded.isapproved,active = excluded.active, lastmodifieddate = excluded.lastmodifieddate, ogr_geometry = excluded.ogr_geometry, ogr_geography=excluded.ogr_geography, modifiedon=excluded.createdon`, data.TenantUID, data.TenantGroupUID, data.GeofenceUID, data.GeofenceNameEN, data.GeofenceNameOL, data.GeofenceTypeID, data.IsApproved, data.Active, data.LastModifiedDate, data.OgrGeometry, currentTime)
		} else {
			batch.Queue(`UPDATE tblmstgeofence SET active = $1, lastmodifieddate = $2, ogr_geometry = NULL, ogr_geography= NULL, modifiedon= $6 WHERE tenantuid = $3 AND tenantgroupuid = $4 AND geofenceuid = $5`, data.Active, data.LastModifiedDate, data.TenantUID, data.TenantGroupUID, data.GeofenceUID, currentTime)
		}

	}

	//constructing area data...
	for _, data := range spatialDataRequestData.AreaData {

		if len(strings.TrimSpace(data.OgrGeometry)) > 0 {
			batch.Queue(`INSERT INTO tblmstarea (tenantuid,tenantgroupuid,areauid,active,lastmodifieddate, ogr_geometry, ogr_geography, createdon)
		VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6,4326), ST_GeographyFromText($6), $7) ON CONFLICT ON CONSTRAINT tblmstarea_pkey DO UPDATE SET active = excluded.active,lastmodifieddate = excluded.lastmodifieddate, ogr_geometry = excluded.ogr_geometry, ogr_geography=excluded.ogr_geography, modifiedon=excluded.createdon`, data.TenantUID, data.TenantGroupUID, data.AreaUID, data.Active, data.LastModifiedDate, data.OgrGeometry, currentTime)
		} else {
			batch.Queue(`UPDATE tblmstarea SET active = $1, lastmodifieddate = $2, ogr_geometry = NULL, ogr_geography= NULL, modifiedon= $6 WHERE tenantuid = $3 AND tenantgroupuid = $4 AND areauid = $5`, data.Active, data.LastModifiedDate, data.TenantUID, data.TenantGroupUID, data.AreaUID, currentTime)
		}

	}

	//constructing zone data...
	for _, data := range spatialDataRequestData.ZoneData {

		if len(strings.TrimSpace(data.OgrGeometry)) > 0 {
			batch.Queue(`INSERT INTO tblmstzone (tenantuid,tenantgroupuid,zoneuid,active,lastmodifieddate, ogr_geometry, ogr_geography, createdon)
		VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6,4326), ST_GeographyFromText($6), $7) ON CONFLICT ON CONSTRAINT tblmstzone_pkey DO UPDATE SET active = excluded.active,lastmodifieddate = excluded.lastmodifieddate, ogr_geometry = excluded.ogr_geometry, ogr_geography=excluded.ogr_geography, modifiedon=excluded.createdon`, data.TenantUID, data.TenantGroupUID, data.ZoneUID, data.Active, data.LastModifiedDate, data.OgrGeometry, currentTime)
		} else {
			batch.Queue(`UPDATE tblmstzone SET active = $1, lastmodifieddate = $2, ogr_geometry = NULL, ogr_geography= NULL , modifiedon= $6 WHERE tenantuid = $3 AND tenantgroupuid = $4 AND zoneuid = $5`, data.Active, data.LastModifiedDate, data.TenantUID, data.TenantGroupUID, data.ZoneUID, currentTime)
		}

	}

	//constructing nogo-area data...
	for _, data := range spatialDataRequestData.NoGoAreaData {

		if len(strings.TrimSpace(data.OgrGeometry)) > 0 {
			batch.Queue(`INSERT INTO tblmstvehiclenogoarea (tenantuid,tenantgroupuid,nogoareageofenceuid,active,lastmodifieddate, ogr_geometry, ogr_geography,createdon)
		VALUES ($1, $2, $3, $4, $5, ST_GeomFromText($6,4326), ST_GeographyFromText($6), $7) ON CONFLICT ON CONSTRAINT tblmstvehiclenogoarea_pkey DO UPDATE SET active = excluded.active,lastmodifieddate = excluded.lastmodifieddate, ogr_geometry = excluded.ogr_geometry, ogr_geography=excluded.ogr_geography, modifiedon=excluded.createdon`, data.TenantUID, data.TenantGroupUID, data.NoGoAreaGeofenceUID, data.Active, data.LastModifiedDate, data.OgrGeometry, currentTime)
		} else {
			batch.Queue(`UPDATE tblmstvehiclenogoarea SET active = $1, lastmodifieddate = $2, ogr_geometry = NULL, ogr_geography= NULL , modifiedon= $6 WHERE tenantuid = $3 AND tenantgroupuid = $4 AND nogoareageofenceuid = $5`, data.Active, data.LastModifiedDate, data.TenantUID, data.TenantGroupUID, data.NoGoAreaGeofenceUID, currentTime)
		}

	}

	//sending batch....
	batchResult := dbc.SendBatch(ctx, batch)

	//executing all geofence based queries....
	for _, data := range spatialDataRequestData.GeofenceData {
		commandTag, err := batchResult.Exec()
		if err != nil {
			logger.Log().Error(fmt.Sprintf("batchResult geofenceUID: %s Error : %v", data.GeofenceUID, err.Error()))

			tx.Rollback(ctx)
			return 0, err
		}

		if commandTag.RowsAffected() > 0 {
			recordAffected = recordAffected + commandTag.RowsAffected()
		}
	}

	//executing all area based queries....
	for _, data := range spatialDataRequestData.AreaData {
		commandTag, err := batchResult.Exec()
		if err != nil {
			logger.Log().Error(fmt.Sprintf("batchResult AreaUID: %s Error : %v", data.AreaUID, err.Error()))

			tx.Rollback(ctx)
			return 0, err
		}
		

		if commandTag.RowsAffected() > 0 {
			recordAffected = recordAffected + commandTag.RowsAffected()
		}
	}

	//executing all zone based queries....
	for _, data := range spatialDataRequestData.ZoneData {
		commandTag, err := batchResult.Exec()
		if err != nil {
			logger.Log().Error(fmt.Sprintf("batchResult zoneuid: %s Error : %v", data.ZoneUID, err.Error()))

			tx.Rollback(ctx)
			return 0, err
		}

		if commandTag.RowsAffected() > 0 {
			recordAffected = recordAffected + commandTag.RowsAffected()
		}
	}

	//executing all Nogo-Area based queries....
	for _, data := range spatialDataRequestData.NoGoAreaData {
		commandTag, err := batchResult.Exec()
		if err != nil {
			logger.Log().Error(fmt.Sprintf("batchResult NoGoAreaUID: %s Error : %v", data.NoGoAreaGeofenceUID, err.Error()))

			tx.Rollback(ctx)
			return 0, err
		}

		if commandTag.RowsAffected() > 0 {
			recordAffected = recordAffected + commandTag.RowsAffected()
		}
	}

	//closing current batch result..
	batchResult.Close()

	//committing trans
	if recordAffected > 0 {
		tx.Commit(ctx)
	}

	return recordAffected, nil
}

//Close conn ...
func Close() {
	dbc.Close(context.Background())
}
