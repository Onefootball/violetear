package violetear

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func expectDeepEqual(t *testing.T, a interface{}, b interface{}) {
	if !reflect.DeepEqual(a, b) {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

type testRouter struct {
	path     string
	methods  string
	requests []testRequests
}

type testRequests struct {
	request string
	method  string
	expect  int
}

type testDynamicRoutes struct {
	name  string
	regex string
}

var dynamicRoutes = []testDynamicRoutes{
	{":uuid", `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`},
	{":ip", `^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`},
}

var routes = []testRouter{
	{"/", "", []testRequests{
		{"/", "GET", 200},
	}},
	{"*", "GET", []testRequests{
		{"/a", "GET", 200},
		{"/a", "HEAD", 405},
		{"/a", "POST", 405},
	}},
	{"/:uuid", "GET, HEAD", []testRequests{
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25", "GET", 200},
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25", "HEAD", 200},
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25", "POST", 405},
	}},
	{"/:uuid/1/", "PUT", []testRequests{
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25/1", "PUT", 200},
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25/2", "GET", 404},
		{"/3B96853C-EF0B-44BC-8820-A982A5756E25/not_found/44", "GET", 404},
		{"/D0ABD486-B05A-436B-BBD1-E320CDC87916/1", "PUT", 200},
	}},
	{"/root", "GET,HEAD", []testRequests{
		{"/root", "GET", 200},
		{"/root", "HEAD", 200},
		{"/root", "OPTIONS", 405},
		{"/root", "POST", 405},
		{"/root", "PUT", 405},
	}},
	{"/root/:ip/", "GET", []testRequests{
		{"/root/10.0.0.0", "GET", 200},
		{"/root/172.16.0.0", "GET", 200},
		{"/root/192.168.0.1", "GET", 200},
		{"/root/300.0.0.0", "GET", 404},
	}},
	{"/root/:ip/aaa/", "GET", []testRequests{}},
	{"/root/:ip/aaa/:uuid", "GET", []testRequests{}},
	{"/root/:uuid/", "PATCH", []testRequests{
		{"/root/3B96853C-EF0B-44BC-8820-A982A5756E25", "GET", 405},
		{"/root/3B96853C-EF0B-44BC-8820-A982A5756E25", "PATCH", 200},
	}},
	{"/root/:uuid/-/:uuid", "GET", []testRequests{
		{"/root/22314BF-4A90-46C8-948D-5507379BD0DD/-/4293C253-6C7E-4B01-90F2-18203FAB2AEC", "GET", 404},
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/-/4293C253-6C7E-4B01-90F2-18203FAB2AE", "GET", 404},
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/-/4293C253-6C7E-4B01-90F2-18203FAB2AEF", "GET", 200},
		{"/root/E22314BF-4A90-46C8-948D-5507379BD0DD/-/4293C253-6C7E-4B01-90F2-18203FAB2AEC", "GET", 200},
	}},
	{"/root/:uuid/:uuid", "", []testRequests{
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AE", "GET", 404},
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF", "GET", 200},
	}},
	{"/root/:uuid/:uuid/end", "GET", []testRequests{
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/end", "GET", 200},
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/end-not-found", "GET", 404},
	}},
	{"/toor/", "GET", []testRequests{
		{"/toor", "GET", 200},
	}},
	{"/toor/aaa", "GET", []testRequests{
		{"/toor/aaa", "GET", 200},
		{"/toor/abc", "GET", 404},
	}},
	{"/toor/*", "GET", []testRequests{
		{"/toor/abc", "GET", 200},
		{"/toor/epazote", "GET", 200},
		{"/toor/naranjas", "GET", 200},
	}},
	{"/toor/1/2", "GET", []testRequests{
		{"/toor/1/2", "GET", 200},
	}},
	{"/toor/1/*", "GET", []testRequests{
		{"/toor/1/catch-me", "GET", 200},
		{"/toor/1/catch-me/too", "GET", 200},
		{"/toor/1/catch-me/too/foo/bar", "GET", 200},
	}},
	{"/toor/1/2/3", "GET", []testRequests{
		{"/toor/1/2/3", "GET", 200},
	}},
	{"/not-found", "GET", []testRequests{
		{"/toor/1/2/3/4", "GET", 404},
		{"catch_me", "GET", 200},
	}},
	{"/root/:uuid/:uuid/:ip/catch-me", "GET", []testRequests{}},
	{"/root/:uuid/:uuid/:ip/catch-me/*", "GET", []testRequests{}},
	{"/root/:uuid/:uuid/:ip/dont-wcatch-me", "GET", []testRequests{}},
	{"/root/:uuid/:uuid/:ip/dont-wcatch-me", "GET", []testRequests{}},
	{"/root/:uuid/:uuid/:ip/", "GET", []testRequests{
		{"/root/122314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8", "GET", 200},
		{"/root/122314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8/catch-me", "GET", 200},
		{"/root/122314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8/catch-me/also", "GET", 200},
		{"/root/122314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8/catch-me/also/a/b/c", "GET", 200},
		{"/root/122314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8/dont-catch-me", "GET", 404},
		{"/root/A22314BF-4A90-46C8-948D-5507379BD0DD/4293C253-6C7E-4B01-90F2-18203FAB2AEF/8.8.8.8", "GET", 200},
	}},
	{"/violetear/:ip/:uuid", "GET", []testRequests{
		{"/violetear/", "GET", 404},
		{"/violetear/127.0.0.1/", "GET", 404},
		{"/violetear/127.0.0.1/A22314BF-4A90-46C8-948D-5507379BD0DD/", "GET", 200},
		{"/violetear/127.0.0.1/A22314BF-4A90-46C8-948D-5507379BD0DD/not-found", "GET", 404},
	}},
}

func TestRouter(t *testing.T) {
	router := New()
	router.Verbose = false
	router.SetHeader("X-app-epazote", "1.1")
	router.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/hello", nil)

	router.ServeHTTP(w, req)
	expect(t, w.Code, http.StatusOK)
	expect(t, len(w.HeaderMap), 2)
	expectDeepEqual(t, w.HeaderMap["X-App-Epazote"], []string{"1.1"})
	fmt.Println(w.Body)
}

func TestRoutes(t *testing.T) {
	router := New()
	router.Verbose = false
	for _, v := range dynamicRoutes {
		router.AddRegex(v.name, v.regex)
	}

	for _, v := range routes {
		if len(v.methods) < 1 {
			v.methods = "ALL"
		}
		router.HandleFunc(v.path, func(w http.ResponseWriter, r *http.Request) {}, v.methods)

		var w *httptest.ResponseRecorder

		for _, v := range v.requests {
			w = httptest.NewRecorder()
			req, _ := http.NewRequest(v.method, v.request, nil)
			router.ServeHTTP(w, req)
			expect(t, w.Code, v.expect)
			if w.Code != v.expect {
				log.Fatalf("[%s - %s - %d > %d]", v.request, v.method, v.expect, w.Code)
			}
		}
	}

}