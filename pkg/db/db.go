// Copyright 2017 The OpenSDS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/*
This module implements the database operation of data structure
defined in api module.

*/

package db

import (
	"fmt"
	"strings"

	c "github.com/sodafoundation/telemetry/pkg/context"
	"github.com/sodafoundation/telemetry/pkg/db/drivers/etcd"
	"github.com/sodafoundation/telemetry/pkg/model"
	. "github.com/sodafoundation/telemetry/pkg/utils/config"
	fakedb "github.com/sodafoundation/telemetry/testutils/db"
)

// C is a global variable that controls database module.
var C Client

// Init function can perform some initialization work of different databases.
func Init(db *Database) {
	switch db.Driver {
	case "mysql":
		// C = mysql.Init(db.Driver, db.Crendential)
		fmt.Printf("mysql is not implemented right now!")
		return
	case "etcd":
		C = etcd.NewClient(strings.Split(db.Endpoint, ","))
		return
	case "fake":
		C = fakedb.NewFakeDbClient()
		return
	default:
		fmt.Printf("Can't find database driver %s!\n", db.Driver)
	}
}

// Client is an interface for exposing some operations of managing database
// client.
type Client interface {
	CreateDock(ctx *c.Context, dck *model.DockSpec) (*model.DockSpec, error)

	GetDock(ctx *c.Context, dckID string) (*model.DockSpec, error)

	ListDocks(ctx *c.Context) ([]*model.DockSpec, error)

	ListDocksWithFilter(ctx *c.Context, m map[string][]string) ([]*model.DockSpec, error)

	UpdateDock(ctx *c.Context, dckID, name, desp string) (*model.DockSpec, error)

	DeleteDock(ctx *c.Context, dckID string) error

	GetDockByPoolId(ctx *c.Context, poolId string) (*model.DockSpec, error)

	CreatePool(ctx *c.Context, pol *model.StoragePoolSpec) (*model.StoragePoolSpec, error)

	GetPool(ctx *c.Context, polID string) (*model.StoragePoolSpec, error)

	ListAvailabilityZones(ctx *c.Context) ([]string, error)

	ListPools(ctx *c.Context) ([]*model.StoragePoolSpec, error)

	ListPoolsWithFilter(ctx *c.Context, m map[string][]string) ([]*model.StoragePoolSpec, error)

	UpdatePool(ctx *c.Context, polID, name, desp string, usedCapacity int64, used bool) (*model.StoragePoolSpec, error)

	DeletePool(ctx *c.Context, polID string) error

	CreateProfile(ctx *c.Context, prf *model.ProfileSpec) (*model.ProfileSpec, error)

	GetProfile(ctx *c.Context, prfID string) (*model.ProfileSpec, error)

	GetDefaultProfile(ctx *c.Context) (*model.ProfileSpec, error)

	GetDefaultProfileFileShare(ctx *c.Context) (*model.ProfileSpec, error)

	ListProfiles(ctx *c.Context) ([]*model.ProfileSpec, error)

	ListProfilesWithFilter(ctx *c.Context, m map[string][]string) ([]*model.ProfileSpec, error)

	UpdateProfile(ctx *c.Context, prfID string, input *model.ProfileSpec) (*model.ProfileSpec, error)

	DeleteProfile(ctx *c.Context, prfID string) error

	AddCustomProperty(ctx *c.Context, prfID string, custom model.CustomPropertiesSpec) (*model.CustomPropertiesSpec, error)

	ListCustomProperties(ctx *c.Context, prfID string) (*model.CustomPropertiesSpec, error)

	RemoveCustomProperty(ctx *c.Context, prfID, customKey string) error
}
