// Copyright 2019 The OpenSDS Authors.
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
This module implements a entry into the OpenSDS northbound REST service.

*/

package routers

import (
	"github.com/astaxie/beego"
	bctx "github.com/astaxie/beego/context"
	"github.com/sodafoundation/telemetry/pkg/api/controllers"
	"github.com/sodafoundation/telemetry/pkg/utils/constants"
)

func init() {

	// add router for v1beta api
	ns :=
		beego.NewNamespace("/"+constants.APIVersion,
			beego.NSCond(func(ctx *bctx.Context) bool {
				// To judge whether the scheme is legal or not.
				if ctx.Input.Scheme() != "http" && ctx.Input.Scheme() != "https" {
					return false
				}

				return true
			}),


		)
	beego.AddNamespace(ns)

	// add router for api version
	beego.Router("/", &controllers.VersionPortal{}, "get:ListVersions")
	beego.Router("/:apiVersion", &controllers.VersionPortal{}, "get:GetVersion")
}
