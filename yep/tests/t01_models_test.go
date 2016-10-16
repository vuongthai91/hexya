// Copyright 2016 NDP Systèmes. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tests

import (
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/npiganeau/yep/yep/models"
	_ "github.com/npiganeau/yep/yep/tests/test_module"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateDB(t *testing.T) {

	Convey("Database creation should run fine", t, func() {
		Convey("Bootstrap should not panic", func() {
			So(models.BootStrap, ShouldNotPanic)
		})
	})

	Convey("Truncating all tables...", t, func() {
		db := sqlx.MustConnect(DBARGS.Driver, fmt.Sprintf("dbname=%s sslmode=disable user=%s password=%s",
			DBARGS.DB, DBARGS.User, DBARGS.Password))
		for _, tn := range []string{"test___user", "test___profile", "test___tag", "test___post"} {
			db.MustExec(fmt.Sprintf(`TRUNCATE TABLE "%s" CASCADE`, tn))
		}
		db.Close()
	})
}