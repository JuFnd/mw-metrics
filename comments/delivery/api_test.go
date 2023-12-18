package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/mocks"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/golang/mock/gomock"
)

func createBody(req requests.CommentRequest) io.Reader {
	jsonReq, _ := json.Marshal(req)

	body := bytes.NewBuffer(jsonReq)
	return body
}


func TestComment(t *testing.T) {
	testCases := map[string]struct {
		method string
		params map[string]string
		result requests.Response
	}{
		"Bad method": {
			method: http.MethodPost,
			params: map[string]string{},
			result: requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
		},
		"No Film": {
			method: http.MethodGet,
			params: map[string]string{},
			result: requests.Response{Status: http.StatusBadRequest, Body: nil},
		},
		"Core error": {
			method: http.MethodGet,
			params: map[string]string{"film_id": "0"},
			result: requests.Response{Status: http.StatusInternalServerError, Body: nil},
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetFilmComments(uint64(0), uint64(0), uint64(10)).Return(nil, fmt.Errorf("core_err")).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/comment", nil)
		q := r.URL.Query()
		for key, value := range curr.params {
			q.Add(key, value)
		}
		r.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

        response := requests.Response{Status: http.StatusBadRequest, Body: nil}

		api.Comment(w, r)

		if response.Status != curr.result.Status {
			t.Errorf("unexpected status: %d", response.Status)
			return
		}
		if !reflect.DeepEqual(response.Body, curr.result.Body) {
			t.Errorf("wanted %v, got %v", curr.result.Body, response.Body)
			return
		}
	}
}

func TestCommentAdd(t *testing.T) {
	testCases := map[string]struct {
		method      string
		result      requests.Response
		addCookie   bool
		body        io.Reader
		cookieValue string
	}{
		"Bad method": {
			method:    http.MethodGet,
			result:    requests.Response{Status: http.StatusMethodNotAllowed, Body: nil},
			addCookie: false,
			body:      nil,
		},
		"No cookie": {
			method:    http.MethodPost,
			result:    requests.Response{Status: http.StatusUnauthorized, Body: nil},
			addCookie: false,
			body:      nil,
		},
		"get user id error": {
			method:      http.MethodPost,
			result:      requests.Response{Status: http.StatusInternalServerError, Body: nil},
			addCookie:   true,
			body:        nil,
			cookieValue: "sid1",
		},
		"no body error": {
			method:      http.MethodPost,
			result:      requests.Response{Status: http.StatusBadRequest, Body: nil},
			addCookie:   true,
			body:        nil,
			cookieValue: "sid2",
		},
		"add comment error": {
			method:      http.MethodPost,
			result:      requests.Response{Status: http.StatusInternalServerError, Body: nil},
			addCookie:   true,
			body:        createBody(requests.CommentRequest{Rating: 10, FilmId: 1, Text: ""}),
			cookieValue: "sid2",
		},
		"found error": {
			method:      http.MethodPost,
			result:      requests.Response{Status: http.StatusNotAcceptable, Body: nil},
			addCookie:   true,
			body:        createBody(requests.CommentRequest{Rating: 10, FilmId: 2, Text: ""}),
			cookieValue: "sid2",
		},
		"Ok": {
			method:      http.MethodPost,
			result:      requests.Response{Status: http.StatusOK, Body: nil},
			addCookie:   true,
			body:        createBody(requests.CommentRequest{Rating: 10, FilmId: 3, Text: ""}),
			cookieValue: "sid2",
		},
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockCore := mocks.NewMockICore(mockCtrl)
	mockCore.EXPECT().GetUserId(context.Background(), string("sid1")).Return(uint64(0), fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().GetUserId(context.Background(), string("sid2")).Return(uint64(1), nil).Times(4)
	mockCore.EXPECT().AddComment(uint64(1), uint64(1), uint16(10), string("")).Return(false, fmt.Errorf("core_err")).Times(1)
	mockCore.EXPECT().AddComment(uint64(2), uint64(1), uint16(10), string("")).Return(true, nil).Times(1)
	mockCore.EXPECT().AddComment(uint64(3), uint64(1), uint16(10), string("")).Return(false, nil).Times(1)
	var buff bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buff, nil))

	api := API{core: mockCore, lg: logger}

	response := requests.Response{Status: http.StatusBadRequest, Body: nil}

	for _, curr := range testCases {
		r := httptest.NewRequest(curr.method, "/api/v1/comment/add", curr.body)
		w := httptest.NewRecorder()

		if curr.addCookie {
			r.AddCookie(&http.Cookie{Name: "session_id", Value: curr.cookieValue})
		}
		api.AddComment(w, r)

		if response.Status != curr.result.Status {
			fmt.Println(api.lg)
			t.Errorf("unexpected status: %d, wanted: %d", response.Status, curr.result.Status)
			return
		}
		if response.Body != nil {
			t.Errorf("unexpected body %v", response.Body)
			return
		}
	}
}
