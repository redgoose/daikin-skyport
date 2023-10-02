package daikin_test

import (
	"testing"

	"github.com/h2non/gock"
	"github.com/nbio/st"
	"github.com/redgoose/daikin-skyport/daikin"
)

var urlBase = "https://api.daikinskyport.com"

func TestGetDevices(t *testing.T) {
	defer gock.Off()
	// gock.Observe(gock.DumpRequest)

	email := "test@test.com"
	password := "mypassword"
	accessToken := "foo"

	gock.New(urlBase).
		Post("/users/auth/login").
		MatchType("json").
		JSON(map[string]string{"email": email, "password": password}).
		Reply(200).
		JSON(map[string]string{"accessToken": accessToken})

	gock.New(urlBase).
		Get("/devices").
		MatchHeader("Authorization", accessToken).
		Reply(200).
		BodyString(`[{"id":"0000000-0000-0000-0000-000000000000","locationId":"0000000-0000-0000-0000-000000000000","name":"Main Room","model":"ONEPLUS","firmwareVersion":"3.2.19","createdDate":1691622040,"hasOwner":true,"hasWrite":true}]`)

	d := daikin.New(email, password)
	devices, _ := d.GetDevices()

	t.Log(devices)

	st.Expect(t, (*devices)[0].Id, "0000000-0000-0000-0000-000000000000")
	st.Expect(t, gock.IsDone(), true)
}
