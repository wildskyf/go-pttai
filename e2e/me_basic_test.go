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

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/ailabstw/go-pttai/common"
	"github.com/ailabstw/go-pttai/common/types"
	"github.com/ailabstw/go-pttai/me"
	pkgservice "github.com/ailabstw/go-pttai/service"
	"github.com/stretchr/testify/assert"
	baloo "gopkg.in/h2non/baloo.v3"
)

func TestMeBasic(t *testing.T) {
	NNodes = 1
	isDebug := true

	var bodyString string
	var marshaledID []byte
	assert := assert.New(t)

	setupTest(t)
	defer teardownTest(t)

	t0 := baloo.New("http://127.0.0.1:9450")

	// 1. get
	bodyString = `{"id": "testID", "method": "me_get", "params": []}`

	me0_1 := &me.BackendMyInfo{}

	testCore(t0, bodyString, me0_1, t, isDebug)

	assert.Equal(types.StatusAlive, me0_1.Status)

	// 2. get total weight
	bodyString = `{"id": "testID", "method": "me_getTotalWeight", "params": []}`

	var totalWeigtht0_2 uint32
	testCore(t0, bodyString, &totalWeigtht0_2, t, isDebug)

	assert.Equal(uint32(me.WeightDesktop), totalWeigtht0_2)

	// 3. getRawMe
	bodyString = `{"id": "testID", "method": "me_getRawMe", "params": []}`

	me0_3 := &me.MyInfo{}

	testCore(t0, bodyString, me0_3, t, isDebug)

	assert.Equal(types.StatusAlive, me0_3.Status)
	assert.Equal(me0_1.ID, me0_3.ID)
	assert.Equal(1, len(me0_3.OwnerIDs))
	assert.Equal(me0_3.ID, me0_3.OwnerIDs[0])
	assert.Equal(true, me0_3.IsOwner(me0_3.ID))

	// 4. show-my-key
	bodyString = `{"id": "testID", "method": "me_showMyKey", "params": []}`

	var myKey0_4 string

	testCore(t0, bodyString, &myKey0_4, t, isDebug)
	if isDebug {
		t.Logf("myKey0_4: %v\n", myKey0_4)
	}

	// 4.1 validate-my-key

	bodyString = fmt.Sprintf(`{"id": "testID", "method": "me_validateMyKey", "params": ["%v"]}`, string(myKey0_4))

	var isValid0_4_1 bool

	testCore(t0, bodyString, &isValid0_4_1, t, isDebug)
	assert.Equal(true, isValid0_4_1)

	time.Sleep(5 * time.Second)

	// 6. getJoinKeyInfo
	bodyString = `{"id": "testID", "method": "me_getJoinKeyInfos", "params": []}`

	dataJoinKeyInfos0_6 := &struct {
		Result []*pkgservice.KeyInfo `json:"result"`
	}{}
	testListCore(t0, bodyString, dataJoinKeyInfos0_6, t, isDebug)
	assert.NotEqual(0, len(dataJoinKeyInfos0_6.Result))

	// 7. raft-satus
	marshaledID, _ = me0_3.ID.MarshalText()
	bodyString = fmt.Sprintf(`{"id": "testID", "method": "me_getRaftStatus", "params": ["%v"]}`, string(marshaledID))

	raftStatus0_7 := &me.RaftStatus{}
	testCore(t0, bodyString, raftStatus0_7, t, true)

	assert.Equal(me0_1.RaftID, raftStatus0_7.Lead)
	assert.Equal(me0_1.RaftID, raftStatus0_7.ConfState.Nodes[0])
	assert.Equal(uint32(me.WeightDesktop), raftStatus0_7.ConfState.Weights[0])

	// 8. getOpKeyInfo
	bodyString = `{"id": "testID", "method": "me_getOpKeyInfos", "params": []}`

	dataOpKeyInfos0_8 := &struct {
		Result []*pkgservice.KeyInfo `json:"result"`
	}{}
	testListCore(t0, bodyString, dataOpKeyInfos0_8, t, isDebug)
	assert.Equal(1, len(dataOpKeyInfos0_8.Result))
	opKeyInfo0_8 := dataOpKeyInfos0_8.Result[0]

	// 8.1 ptt.GetOps
	bodyString = `{"id": "testID", "method": "ptt_getOps", "params": []}`

	dataOpKeyInfo0_8_1 := &struct {
		Result map[common.Address]*types.PttID `json:"result"`
	}{}
	testListCore(t0, bodyString, dataOpKeyInfo0_8_1, t, isDebug)

	opKeyInfoMap0_8_1 := dataOpKeyInfo0_8_1.Result
	entityID, ok := opKeyInfoMap0_8_1[*opKeyInfo0_8.Hash]

	assert.Equal(true, ok)
	assert.Equal(me0_3.ID, entityID)

	// 9. MasterOplog
	bodyString = `{"id": "testID", "method": "me_getMasterOplogList", "params": ["", 0, 2]}`

	dataMasterOplogs0_9 := &struct {
		Result []*me.MasterOplog `json:"result"`
	}{}
	testListCore(t0, bodyString, dataMasterOplogs0_9, t, isDebug)
	assert.Equal(1, len(dataMasterOplogs0_9.Result))
	masterOplog0_9 := dataMasterOplogs0_9.Result[0]
	assert.Equal(me0_3.ID[:common.AddressLength], masterOplog0_9.CreatorID[common.AddressLength:])
	assert.Equal(me0_3.ID, masterOplog0_9.ObjID)
	assert.Equal(me.MasterOpTypeAddMaster, masterOplog0_9.Op)
	assert.Equal(nilPttID, masterOplog0_9.PreLogID)
	assert.Equal(types.Bool(true), masterOplog0_9.IsSync)
	assert.Equal(masterOplog0_9.ID, masterOplog0_9.MasterLogID)

	// 9.1. OpKeyOplog
	bodyString = `{"id": "testID", "method": "me_getOpKeyOplogList", "params": ["", 0, 2]}`

	dataOpKeyOplogs0_9_1 := &struct {
		Result []*pkgservice.OpKeyOplog `json:"result"`
	}{}
	testListCore(t0, bodyString, dataOpKeyOplogs0_9_1, t, isDebug)
	assert.Equal(1, len(dataOpKeyOplogs0_9_1.Result))
	opKeyOplog0_9_1 := dataOpKeyOplogs0_9_1.Result[0]
	assert.Equal(me0_3.ID, opKeyOplog0_9_1.CreatorID)
	assert.Equal(opKeyInfo0_8.ID, opKeyOplog0_9_1.ObjID)
	assert.Equal(pkgservice.OpKeyOpTypeCreateOpKey, opKeyOplog0_9_1.Op)
	assert.Equal(nilPttID, opKeyOplog0_9_1.PreLogID)
	assert.Equal(types.Bool(true), opKeyOplog0_9_1.IsSync)
	assert.Equal(masterOplog0_9.ID, opKeyOplog0_9_1.MasterLogID)

}