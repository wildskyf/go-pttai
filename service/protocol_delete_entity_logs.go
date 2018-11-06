// Copyright 2018 The go-pttai Authors
// This file is part of the go-pttai library.
//
// The go-pttai library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-pttai library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-pttai library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"reflect"

	"github.com/ailabstw/go-pttai/common/types"
)

func (pm *BaseProtocolManager) HandleDeleteEntityLog(
	oplog *BaseOplog,
	info ProcessInfo,

	opData OpData,
	status types.Status,

	setLogDB func(oplog *BaseOplog),
	postdelete func(opData OpData) error,
) ([]*BaseOplog, error) {

	entity := pm.Entity()

	err := oplog.GetData(opData)
	if err != nil {
		return nil, err
	}

	// 1. lock object
	err = entity.Lock()
	if err != nil {
		return nil, err
	}
	defer entity.Unlock()

	// 3. already deleted
	origStatus := entity.GetStatus()
	origStatusClass := types.StatusToStatusClass(origStatus)
	if origStatusClass == types.StatusClassDeleted {
		if oplog.UpdateTS.IsLess(entity.GetUpdateTS()) {
			err = EntitySetStatusWithOplog(entity, status, oplog)
			if err != nil {
				return nil, err
			}
		}
		return nil, ErrNewerOplog
	}

	// 4. sync-info
	origSyncInfo := entity.GetSyncInfo()
	if origSyncInfo != nil {
		syncLogID := origSyncInfo.GetLogID()
		if !reflect.DeepEqual(syncLogID, oplog.ID) {
			err = entity.RemoveSyncInfo(oplog, opData, origSyncInfo, info)
			if err != nil {
				return nil, err
			}

			_, err := pm.RemoveNonSyncOplog(setLogDB, syncLogID, true, false)
			if err != nil {
				return nil, err
			}
		}

		entity.SetSyncInfo(nil)
	}

	// 7. saveDeleteObj
	err = EntitySetStatusWithOplog(entity, status, oplog)
	if err != nil {
		return nil, err
	}

	// 7.1
	if postdelete != nil {
		postdelete(opData)
	}

	return nil, nil
}

/**********
 * Handle PendingDeleteEntityLog
 **********/

func (pm *BaseProtocolManager) HandlePendingDeleteEntityLog(
	oplog *BaseOplog, info ProcessInfo,

	status types.Status,
	op OpType,
	opData OpData,

	setLogDB func(oplog *BaseOplog),
) ([]*BaseOplog, error) {

	entity := pm.Entity()

	// 1. lock obj
	err := entity.Lock()
	if err != nil {
		return nil, err
	}
	defer entity.Unlock()

	// 3. already deleted
	origStatus := entity.GetStatus()
	origStatusClass := types.StatusToStatusClass(origStatus)
	if origStatusClass == types.StatusClassDeleted {
		return nil, ErrNewerOplog
	}

	// 4. sync info
	origSyncInfo := entity.GetSyncInfo()
	if origSyncInfo != nil {
		syncLogID := origSyncInfo.GetLogID()
		if !reflect.DeepEqual(syncLogID, oplog.ID) {
			err = entity.RemoveSyncInfo(oplog, opData, origSyncInfo, info)
			if err != nil {
				return nil, err
			}

			_, err := pm.RemoveNonSyncOplog(setLogDB, syncLogID, false, false)
			if err != nil {
				return nil, err
			}
		}

		entity.SetSyncInfo(nil)
	}

	// 5. save obj
	entity.SetPendingDeleteSyncInfo(status, oplog)
	err = entity.Save(true)
	if err != nil {
		return nil, err
	}

	// 6. update delete info
	entity.UpdateDeleteInfo(oplog, info)

	return nil, nil
}

/**********
 * Set Newest DeleteObjectLog
 **********/

func (pm *BaseProtocolManager) SetNewestDeleteEntityLog(
	oplog *BaseOplog,
) (types.Bool, error) {
	return false, nil
}

/**********
 * Handle Failed DeleteEntityLog
 **********/

func (pm *BaseProtocolManager) HandleFailedDeleteEntityLog(
	oplog *BaseOplog,
) error {

	entity := pm.Entity()

	// 1. lock obj
	err := entity.Lock()
	if err != nil {
		return err
	}
	defer entity.Unlock()

	// 3. check validity
	syncInfo := entity.GetSyncInfo()
	if syncInfo == nil || !reflect.DeepEqual(syncInfo.GetLogID(), oplog.ID) {
		return nil
	}

	if oplog.UpdateTS.IsLess(syncInfo.GetUpdateTS()) {
		return nil
	}

	// 4. handle fails
	entity.SetSyncInfo(nil)
	err = entity.Save(true)
	if err != nil {
		return err
	}

	return nil
}
