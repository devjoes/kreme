package parsejwt

import (
	"bytes"
	"math/rand"
	"net/http"
	"testing"
	"text/template"
	"time"

	"github.com/devjoes/kreme/pkg/datasources"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestFoo(t *testing.T) {
	temp, _ := template.New("test").Parse("{{.Context.someOtherDs.Foo.Token}}")
	context := map[string]interface{}{"someOtherDs": struct{ Foo struct{ Token string } }{Foo: struct{ Token string }{Token: "bar"}}}

	x := ParseJwt{context: &context}
	var buf bytes.Buffer
	err := temp.Execute(&buf, x)
	assert.Nil(t, err)
	str := buf.String()
	t.Log(str)
}

func testTokenRetrival(req *http.Request, context *map[string]interface{}, tokenTemplate string, key []byte) func(t *testing.T) {
	return func(t *testing.T) {
		parseJwt := NewParseJwt(&ParseJwtOptions{
			ErrorIfTokenMissing: true,
			TokenTemplate:       tokenTemplate,
			SigningSecret:       key,
		})
		helper := datasources.NewDataSourceHelper(datasources.DataSourceInfo{DataSource: &parseJwt}, context, req)
		cacheKey, cacheFor, err := parseJwt.Setup(req, context, &helper)
		assert.NotEqual(t, "", cacheKey)
		assert.Less(t, float64(28), cacheFor.Minutes())
		assert.Greater(t, float64(31), cacheFor.Minutes())
		assert.Nil(t, err)
		result, err := parseJwt.GetData()
		assert.Nil(t, err)
		assert.NotNil(t, result)
	}
}

func TestParsesToken(t *testing.T) {
	key := getRandomBytes(1024)
	req, _ := http.NewRequest("GET", "/foo", nil)
	token := getTestToken(key)
	req.Header.Add("Authorization", "Bearer, "+token)
	t.Run("From header", testTokenRetrival(req, &map[string]interface{}{}, "", key))

	req, _ = http.NewRequest("GET", "/foo", nil)
	context := map[string]interface{}{"someOtherDs": struct{ Foo struct{ Token string } }{Foo: struct{ Token string }{Token: token}}}
	t.Run("From context", testTokenRetrival(req, &context, "{{ .Context.someOtherDs.Foo.Token }}", key))
}
func TestErrorsIfTokenMissing(t *testing.T) {
	req, _ := http.NewRequest("GET", "/foo", nil)
	testTokenRetrival(req, &map[string]interface{}{}, "", []byte{})

	parseJwt := NewParseJwt(&ParseJwtOptions{ErrorIfTokenMissing: true, TokenTemplate: ""})
	helper := datasources.NewDataSourceHelper(datasources.DataSourceInfo{DataSource: &parseJwt}, nil, req)
	_, _, err := parseJwt.Setup(req, nil, &helper)
	assert.NotNil(t, err)
}

func getTestToken(hmacSecret []byte) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"foo": "bar",
		"nbf": time.Now().UTC().Add(-30 * time.Minute).Unix(),
		"exp": time.Now().UTC().Add(30 * time.Minute).Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	tokenString, _ := token.SignedString(hmacSecret)
	return tokenString
}

func getRandomBytes(length int) []byte {
	output := make([]byte, length)
	rand.Read(output)
	return output
}
