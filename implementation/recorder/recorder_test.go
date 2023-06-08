package aurestrecorder

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestConstructFilenameLong(t *testing.T) {
	requestUrl := "https://some.super.long.server.name.that.hopefully.does.not.exist/api/rest/v1/v2/v3/v4/this/is/intentionally/very/very/very/very/long/djkfjhdalsfhdsahjflkdjsahfkjlsdhafjkdshafkjlsdahf/asdfjkldsahfkjlfad/dskjfhjkdsfahlk/sdafjkhsdafklhreuih/dfgjgkjlhgjlkhg?asjdfhlkhewuirfhkjsdhlk=kjhrelrukihsjd&fsdkjhfdjklhsdf=werjkyewuiryuweiry&sfuyfddsjkhjkldsfhldkfs=sdjhdflksjhfdhsf"
	actual, err := ConstructFilenameWithBody("GET", requestUrl, nil)
	expected := "request_get_%2Fapi%2Frest%2Fv1%2Fv2%2Fv3%2Fv4%2Fthis%2Fis%2Fintentionally%2Fvery%2Fvery%2Fvery%2Fvery%2Flong%2Fdjkfjhdalsfhdsahjflkd_fb2e3656d88910ffc49023f99f5e0df6.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructFilenameShort(t *testing.T) {
	requestUrl := "https://some.short.server.name/api/rest/v1/kittens"
	actual, err := ConstructFilenameWithBody("GET", requestUrl, nil)
	expected := "request_get_%2Fapi%2Frest%2Fv1%2Fkittens_d41d8cd98f00b204e9800998ecf8427e.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructFilenameV2Long(t *testing.T) {
	requestUrl := "https://some.super.long.server.name.that.hopefully.does.not.exist/api/rest/v1/v2/v3/v4/this/is/intentionally/very/very/very/very/long/djkfjhdalsfhdsahjflkdjsahfkjlsdhafjkdshafkjlsdahf/asdfjkldsahfkjlfad/dskjfhjkdsfahlk/sdafjkhsdafklhreuih/dfgjgkjlhgjlkhg?asjdfhlkhewuirfhkjsdhlk=kjhrelrukihsjd&fsdkjhfdjklhsdf=werjkyewuiryuweiry&sfuyfddsjkhjkldsfhldkfs=sdjhdflksjhfdhsf"
	actual, err := ConstructFilenameV2WithBody("GET", requestUrl, nil)
	expected := "request_get_api-rest-v1-v2-v3-v4-this-is-intentionally-very-very-very-very-long-djkfjhdalsfhdsahjflkd_fb2e3656.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructFilenameV2Short(t *testing.T) {
	requestUrl := "https://some.short.server.name/api/rest/v1/kittens"
	actual, err := ConstructFilenameV2WithBody("GET", requestUrl, nil)
	expected := "request_get_api-rest-v1-kittens_d41d8cd9.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructFilenameV3Long(t *testing.T) {
	requestUrl := "https://some.super.long.server.name.that.hopefully.does.not.exist/api/rest/v1/v2/v3/v4/this/is/intentionally/very/very/very/very/long/djkfjhdalsfhdsahjflkdjsahfkjlsdhafjkdshafkjlsdahf/asdfjkldsahfkjlfad/dskjfhjkdsfahlk/sdafjkhsdafklhreuih/dfgjgkjlhgjlkhg?asjdfhlkhewuirfhkjsdhlk=kjhrelrukihsjd&fsdkjhfdjklhsdf=werjkyewuiryuweiry&sfuyfddsjkhjkldsfhldkfs=sdjhdflksjhfdhsf"
	actual, err := ConstructFilenameV3WithBody("GET", requestUrl, nil)
	expected := "request_get_api-rest-v1-v2-v3-v4-this-is-intentionally-very-very-very-very-long-djkfjhdalsfhdsahjflkd_fb2e3656.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructFilenameV3Short(t *testing.T) {
	requestUrl := "https://some.short.server.name/api/rest/v1/kittens"
	actual, err := ConstructFilenameV3WithBody("GET", requestUrl, nil)
	expected := "request_get_api-rest-v1-kittens_d41d8cd9.json"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestConstructEqualFilenameV3ForDifferentQueryParameterOrder(t *testing.T) {
	requestUrl1 := "https://some.short.server.name/api/rest/v1/kittens?a=123&z=o&v=666"
	actual1, _ := ConstructFilenameV3WithBody("GET", requestUrl1, nil)

	requestUrl2 := "https://some.short.server.name/api/rest/v1/kittens?z=o&v=666&a=123"
	actual2, _ := ConstructFilenameV3WithBody("GET", requestUrl2, nil)

	require.Equal(t, actual1, actual2)
}
