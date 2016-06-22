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

package models

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestConditions(t *testing.T) {
	Convey("Testing SQL building for conditions", t, func() {
		env := NewEnvironment(1)
		if DBARGS.Driver == "postgres" {
			rs := env.Pool("User")
			rs = rs.Filter("profile_id.best_post_id.title", "=", "foo")
			fields := []string{"user_name", "profile_id.best_post_id.title"}
			sql, args := rs.query.selectQuery(fields)
			So(sql, ShouldEqual, `SELECT "user".user_name, "user__profile__post".title FROM "user" "user" LEFT JOIN "profile" "user__profile" ON "user".profile_id="user__profile".id LEFT JOIN "post" "user__profile__post" ON "user__profile".best_post_id="user__profile__post".id  WHERE "user__profile__post".title = ? `)
			So(args, ShouldContain, "foo")

			rs = env.Pool("User")
			rs = rs.Filter("Profile.BestPost.Title", "=", "foo")
			fields = []string{"UserName", "Profile.BestPost.Title"}
			sql, args = rs.query.selectQuery(fields)
			So(sql, ShouldEqual, `SELECT "user".user_name, "user__profile__post".title FROM "user" "user" LEFT JOIN "profile" "user__profile" ON "user".profile_id="user__profile".id LEFT JOIN "post" "user__profile__post" ON "user__profile".best_post_id="user__profile__post".id  WHERE "user__profile__post".title = ? `)
			So(args, ShouldContain, "foo")

			rs = rs.Filter("Profile.Age", ">=", 12)
			sql, args = rs.query.sqlWhereClause()
			So(sql, ShouldEqual, `WHERE "user__profile__post".title = ? AND "user__profile".age >= ? `)
			So(args, ShouldContain, 12)
			So(args, ShouldContain, "foo")

			c2 := NewCondition().And("user_name", "like", "jane").Or("Profile.Money", "<", 1234.56)
			rs = rs.SetCond(c2)
			sql, args = rs.query.sqlWhereClause()
			So(sql, ShouldEqual, `WHERE "user__profile__post".title = ? AND "user__profile".age >= ? AND ("user".user_name LIKE %?% OR "user__profile".money < ? ) `)
			So(args, ShouldContain, "jane")
			So(args, ShouldContain, 1234.56)

			sql, args = rs.query.selectQuery(fields)
			So(sql, ShouldEqual, `SELECT "user".user_name, "user__profile__post".title FROM "user" "user" LEFT JOIN "profile" "user__profile" ON "user".profile_id="user__profile".id LEFT JOIN "post" "user__profile__post" ON "user__profile".best_post_id="user__profile__post".id  WHERE "user__profile__post".title = ? AND "user__profile".age >= ? AND ("user".user_name LIKE %?% OR "user__profile".money < ? ) `)
		}
		env.cr.Rollback()
	})
}
