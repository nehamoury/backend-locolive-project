package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"privacy-social-backend/internal/repository/db"
	mockdb "privacy-social-backend/internal/repository/mock"
	"privacy-social-backend/internal/util"
)

type eqCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}

	err := util.CheckPassword(e.password, arg.PasswordHash)
	if err != nil {
		return false
	}

	e.arg.PasswordHash = arg.PasswordHash
	return reflect.DeepEqual(e.arg, arg)
}

func (e eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, password}
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCreateUser(t *testing.T) {
	user, password := randomUser(t)

	testCases := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"phone":     user.Phone,
				"email":     user.Email.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username:          user.Username,
					FullName:          user.FullName,
					Phone:             user.Phone,
					Email:             user.Email,
					Provider:          "local",
					IsProfileComplete: true,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, password)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Session{ID: uuid.New()}, nil)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, rec.Code)
				requireBodyMatchUser(t, rec.Body, user)
			},
		},
		{
			name: "InvalidUsername",
			body: gin.H{
				"username":  "invalid#user",
				"password":  password,
				"full_name": user.FullName,
				"phone":     user.Phone,
				"email":     user.Email.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "DuplicateUsername",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"phone":     user.Phone,
				"email":     user.Email.String,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrNoRows) // Just an error to simulate duplicate
			},
			checkResponse: func(rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)

			// Username service calls GetReservedUsernames during NewServer initialization
			store.EXPECT().
				GetReservedUsernames(gomock.Any()).
				Return([]string{}, nil).
				AnyTimes()

			// Username validation calls during user creation flow
			store.EXPECT().
				IsUsernameReserved(gomock.Any(), gomock.Any()).
				Return(false, nil).
				AnyTimes()

			store.EXPECT().
				CheckUsernameExists(gomock.Any(), gomock.Any()).
				Return(false, nil).
				AnyTimes()

			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			request, err := http.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func randomUser(t *testing.T) (user db.User, password string) {
	password = util.RandomString(6)
	hashedPassword, err := util.HashPassword(password)
	require.NoError(t, err)

	user = db.User{
		Username:     util.RandomOwner(),
		PasswordHash: hashedPassword,
		FullName:     util.RandomString(10),
		Phone:        util.RandomString(10),
		Email:        sql.NullString{String: util.RandomEmail(), Valid: true},
	}
	return
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, user db.User) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	type userResponse struct {
		Username  string `json:"username"`
		FullName  string `json:"full_name"`
		Bio       string `json:"bio"`
		AvatarUrl string `json:"avatar_url"`
	}

	var got struct {
		User userResponse `json:"user"`
	}
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)
	require.Equal(t, user.Username, got.User.Username)
	require.Equal(t, user.FullName, got.User.FullName)
	// Password is not returned, so we can't check it directly here, but schema validation covers it.
}
