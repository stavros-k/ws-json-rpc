package api

import (
	"net/http"
	"time"
	"ws-json-rpc/backend/pkg/apitypes"
	"ws-json-rpc/backend/pkg/router"
	"ws-json-rpc/backend/pkg/types"
	"ws-json-rpc/backend/pkg/utils"

	"github.com/go-chi/chi/v5"
)

func (s *Server) GetTeam(w http.ResponseWriter, r *http.Request) error {
	teamID := chi.URLParam(r, "teamID")

	RespondJSON(w, r, http.StatusOK, apitypes.GetTeamResponse{TeamID: teamID, Users: []apitypes.User{{UserID: "Asdf"}}})
	return nil
}

func RegisterGetTeam(path string, rb *router.RouteBuilder, s *Server) {
	rb.MustGet(path, router.RouteSpec{
		OperationID: "getTeam",
		Summary:     "Get a team",
		Description: "Get a team by its ID",
		Group:       TeamGroup,
		Deprecated:  "Use GetTeamResponseV2 instead.",
		Handler:     ErrorHandler(s.GetTeam),
		RequestType: &router.RequestBodySpec{
			Type: apitypes.GetTeamRequest{TeamID: "123"},
			Examples: map[string]any{
				"example-1": apitypes.GetTeamResponse{TeamID: "abxc", Users: []apitypes.User{{UserID: "Asdf"}}},
			},
		},
		Parameters: map[string]router.ParameterSpec{
			"teamID": {
				In:          "path",
				Description: "ID of the team to get",
				Required:    true,
				Type:        new(string),
			},
		},
		Responses: GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				Examples: map[string]any{
					"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
			201: {
				Description: "Successful ping response",
				Type:        apitypes.GetTeamResponse{TeamID: "123", Users: []apitypes.User{{UserID: "123", Name: "John"}}},
				Examples: map[string]any{
					"example-1": apitypes.GetTeamResponse{TeamID: "123", Users: []apitypes.User{{UserID: "123", Name: "John"}}},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}},
				Examples: map[string]any{
					"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}},
				},
			},
		}),
	})
}

func (s *Server) CreateTeam(w http.ResponseWriter, r *http.Request) error {
	RespondJSON(w, r, http.StatusOK, apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK})
	return nil
}

func RegisterCreateTeam(path string, rb *router.RouteBuilder, s *Server) {
	rb.MustPost(path, router.RouteSpec{
		OperationID: "createTeam",
		Summary:     "Create a team",
		Description: "Create a team by its name",
		Group:       TeamGroup,
		Handler:     ErrorHandler(s.CreateTeam),
		RequestType: &router.RequestBodySpec{
			Type: apitypes.CreateTeamRequest{Name: "My Team"},
			Examples: map[string]any{
				"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
			},
		},
		Responses: GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				Examples: map[string]any{
					"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}},
				Examples: map[string]any{
					"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}, URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
				},
			},
		}),
	})
}

func (s *Server) DeleteTeam(w http.ResponseWriter, r *http.Request) error {
	RespondJSON(w, r, http.StatusOK, apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK})
	return nil
}

func RegisterDeleteTeam(path string, rb *router.RouteBuilder, s *Server) {
	rb.MustDelete(path, router.RouteSpec{
		OperationID: "deleteTeam",
		Summary:     "Create a team",
		Description: "Create a team by its name",
		Group:       TeamGroup,
		Handler:     ErrorHandler(s.DeleteTeam),
		RequestType: &router.RequestBodySpec{
			Type: apitypes.CreateTeamRequest{Name: "My Team"},
			Examples: map[string]any{
				"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
			},
		},
		Responses: GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				Examples: map[string]any{
					"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}},
				Examples: map[string]any{
					"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}, URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
				},
			},
		}),
	})

}

func (s *Server) PutTeam(w http.ResponseWriter, r *http.Request) error {
	RespondJSON(w, r, http.StatusOK, apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK})
	return nil
}

func RegisterPutTeam(path string, rb *router.RouteBuilder, s *Server) {
	rb.MustPut(path, router.RouteSpec{
		OperationID: "putTeam",
		Summary:     "Create a team",
		Description: "Create a team by its name",
		Group:       TeamGroup,
		Handler:     ErrorHandler(s.PutTeam),
		RequestType: &router.RequestBodySpec{
			Type: apitypes.CreateTeamRequest{Name: "My Team"},
			Examples: map[string]any{
				"example-1": apitypes.CreateTeamRequest{Name: "My Team"},
			},
		},
		Responses: GenerateResponses(map[int]router.ResponseSpec{
			200: {
				Description: "Successful ping response",
				Type:        apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				Examples: map[string]any{
					"example-1": apitypes.PingResponse{Message: "Pong", Status: apitypes.PingStatusOK},
				},
			},
			400: {
				Description: "Invalid request",
				Type:        apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}},
				Examples: map[string]any{
					"example-1": apitypes.CreateUserResponse{UserID: "123", CreatedAt: time.Time{}, URL: utils.Ptr(types.MustNewURL("https://localhost:8080/user"))},
				},
			},
		}),
	})

}
