package config_test

import (
	"bytes"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/appbricks/cloud-builder/config"
)

var _ = Describe("Auth Context", func() {

	It("loads and saves the auth context", func() {

		authContext := config.NewAuthContext()
		err := authContext.Load(strings.NewReader(jsonToken))
		Expect(err).ToNot(HaveOccurred())

		token := authContext.GetToken()
		Expect(token.AccessToken).To(Equal("MJUYMWZIOGMTMTFLYY0ZNTE2LWIXNTATZTMWYJM3MZVJODRH"))
		Expect(token.TokenType).To(Equal("Bearer"))
		Expect(token.RefreshToken).To(Equal("MZC5OWUZYTCTY2QZNC01MWE3LWE4ZJGTMJKXYZLIMGJJZGU1"))										                    
		Expect(token.Extra("id_token")).To(Equal("JASIUHQQWHKJHASKJHASDIUAHDQIUDHQKDHASKDHCASKHASA"))

		expiry, _ := time.Parse(time.RFC3339, "2021-01-21T21:56:20.457563-05:00")
		Expect(token.Expiry).To(Equal(expiry))

		var buffer bytes.Buffer
		err = authContext.Save(&buffer)
		Expect(err).ToNot(HaveOccurred())
		Expect(strings.TrimSuffix(buffer.String(), "\n")).To(Equal(jsonToken))
	})
})

const jsonToken = `{"token":{"access_token":"MJUYMWZIOGMTMTFLYY0ZNTE2LWIXNTATZTMWYJM3MZVJODRH","token_type":"Bearer","refresh_token":"MZC5OWUZYTCTY2QZNC01MWE3LWE4ZJGTMJKXYZLIMGJJZGU1","expiry":"2021-01-21T21:56:20.457563-05:00"},"tokenExtra":{"id_token":"JASIUHQQWHKJHASKJHASDIUAHDQIUDHQKDHASKDHCASKHASA"}}`
