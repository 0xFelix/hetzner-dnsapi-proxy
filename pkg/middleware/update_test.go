package middleware_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	// "time" // No longer directly used in this file after refactoring time-based logic if any

	// Original package under test
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/config"
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/hetzner" // Assuming this is the correct import for Hetzner client types
	"github.com/0xfelix/hetzner-dnsapi-proxy/pkg/middleware"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// mockHetznerAPIServer (refactored for Ginkgo)
type mockHetznerAPIServer struct {
	server   *httptest.Server
	mu       sync.Mutex
	handlers map[string]http.HandlerFunc
	cfgToken string // Expected API token
	// failFunc func(message string, callerSkip ...int) // To replace t.Errorf, t.Fatalf
}

func newMockHetznerAPIServer(apiToken string) *mockHetznerAPIServer {
	mock := &mockHetznerAPIServer{handlers: make(map[string]http.HandlerFunc), cfgToken: apiToken}
	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		defer mock.mu.Unlock()

		// Check common headers
		if token := r.Header.Get("Auth-API-Token"); token != mock.cfgToken {
			// Use Ginkgo's Fail for errors in mock server logic during tests
			Fail(fmt.Sprintf("Mock API: Auth-API-Token header mismatch: got %q, want %q for %s %s", token, mock.cfgToken, r.Method, r.URL.Path))
			http.Error(w, "Unauthorized - bad token", http.StatusUnauthorized)
			return
		}
		if r.Method == "POST" || r.Method == "PUT" {
			if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
				Fail(fmt.Sprintf("Mock API: Content-Type header mismatch: got %q, want %q for %s %s", contentType, "application/json", r.Method, r.URL.Path))
				http.Error(w, "Bad Request - bad content type", http.StatusBadRequest)
				return
			}
		}

		key := r.Method + " " + r.URL.Path
		// More specific key for GET /records with zone_id to avoid conflicts if not specific enough
		if r.Method == "GET" && r.URL.Path == "/records" && r.URL.Query().Get("zone_id") != "" {
			key = fmt.Sprintf("GET /records?zone_id=%s", r.URL.Query().Get("zone_id"))
		}
		
		if handler, ok := mock.handlers[key]; ok {
			handler(w, r)
			return
		}

		// Fallback for general GET /records if specific zone_id handler wasn't found/set
		// This might be needed if a test sets a general "/records" handler and expects it to be hit
		// However, the previous logic prioritized specific zone_id matches for /records.
		// For clarity, ensure handlers are set for the exact patterns expected.
		// The previous fallback for GET /records based on query param was a bit complex.
		// Simpler approach: ensure tests set specific handlers.

		Fail(fmt.Sprintf("Mock API: Unhandled request: %s %s (Query: %s)", r.Method, r.URL.Path, r.URL.RawQuery))
		http.Error(w, fmt.Sprintf("Unhandled API mock request for %s %s", r.Method, r.URL.Path), http.StatusNotImplemented)
	}))
	return mock
}

func (m *mockHetznerAPIServer) Close() {
	m.server.Close()
}

func (m *mockHetznerAPIServer) URL() string {
	return m.server.URL
}

func (m *mockHetznerAPIServer) SetHandler(method, path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// If path for GET /records includes a zone_id query, format it consistently for lookup
	if method == "GET" && strings.HasPrefix(path, "/records?") {
		// This ensures that if a test sets "/records?zone_id=XYZ", it's stored and looked up correctly.
		// The main dispatcher now also creates keys like this if zone_id is present.
	}
	m.handlers[method+" "+path] = handler
}

// Helper for mock JSON responses (refactored for Ginkgo)
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		err := json.NewEncoder(w).Encode(data)
		Expect(err).NotTo(HaveOccurred(), "Mock API: Failed to write JSON response")
	}
}

// Helper to create ReqData (assuming middleware.ReqData is now exported)
func newTestReqData(name, zone, value, recordType string) *middleware.ReqData {
	return &middleware.ReqData{
		FullName: fmt.Sprintf("%s.%s", name, zone),
		Name:     name,
		Zone:     zone,
		Value:    value,
		Type:     recordType,
	}
}

var _ = Describe("NewUpdater Middleware", func() {
	var (
		mockAPI           *mockHetznerAPIServer
		cfg               *config.Config
		rr                *httptest.ResponseRecorder
		req               *http.Request
		updaterMiddleware http.Handler // This is func(http.Handler) http.Handler
		finalHandler      http.Handler // This is the result of updaterMiddleware(dummyNext)
		nextHandlerCalled bool
		apiToken          string = "test-hetzner-api-token"
		dummyNext         http.HandlerFunc
	)

	BeforeEach(func() {
		mockAPI = newMockHetznerAPIServer(apiToken)
		// Default config for each test
		cfg = &config.Config{
			Token:   apiToken,
			BaseURL: mockAPI.URL(),
			Timeout: 5, // Using original test timeout
		}
		rr = httptest.NewRecorder()
		nextHandlerCalled = false
		dummyNext = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			nextHandlerCalled = true
			w.WriteHeader(http.StatusOK)
		})
		updaterMiddleware = middleware.NewUpdater(cfg) // NewUpdater is from the original package
		finalHandler = updaterMiddleware(dummyNext)
	})

	AfterEach(func() {
		mockAPI.Close()
	})

	Context("when a record needs to be updated (update flow)", func() {
		var reqData *middleware.ReqData
		var zoneID, recordID string

		BeforeEach(func() {
			reqData = newTestReqData("myrecord", "example.com", "192.168.0.1", "A")
			zoneID = "zone123"
			recordID = "rec456"

			// Mock API responses for update flow
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Query().Get("name")).To(Equal(reqData.Zone))
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{
					Zones: []hetzner.Zone{{ID: zoneID, Name: reqData.Zone}},
				})
			})
			mockAPI.SetHandler("GET", fmt.Sprintf("/records?zone_id=%s", zoneID), func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.RecordsResponse{
					Records: []hetzner.Record{
						{ID: recordID, ZoneID: zoneID, Name: reqData.Name, Value: "192.168.0.0", Type: reqData.Type},
					},
				})
			})
			mockAPI.SetHandler("PUT", fmt.Sprintf("/records/%s", recordID), func(w http.ResponseWriter, r *http.Request) {
				var updateReq hetzner.RecordRequest
				err := json.NewDecoder(r.Body).Decode(&updateReq)
				Expect(err).NotTo(HaveOccurred())
				Expect(updateReq.ZoneID).To(Equal(zoneID))
				Expect(updateReq.Name).To(Equal(reqData.Name))
				Expect(updateReq.Type).To(Equal(reqData.Type))
				Expect(updateReq.Value).To(Equal(reqData.Value))
				writeJSONResponse(w, http.StatusOK, hetzner.RecordResponse{Record: hetzner.Record{
					ID: recordID, ZoneID: zoneID, Name: reqData.Name, Value: reqData.Value, Type: reqData.Type,
				}})
			})

			// Prepare request with context
			ctx := middleware.NewContextWithReqData(context.Background(), reqData) // Uses exported func
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})

		It("should successfully update the record and call the next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(nextHandlerCalled).To(BeTrue())
		})
	})

	Context("when a record needs to be created (create flow)", func() {
		var reqData *middleware.ReqData
		var zoneID, newRecordID string
		
		BeforeEach(func() {
			reqData = newTestReqData("newrecord", "example.com", "10.0.0.1", "A")
			zoneID = "zone123"
			newRecordID = "recNew789"

			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Query().Get("name")).To(Equal(reqData.Zone))
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{
					Zones: []hetzner.Zone{{ID: zoneID, Name: reqData.Zone}},
				})
			})
			mockAPI.SetHandler("GET", fmt.Sprintf("/records?zone_id=%s", zoneID), func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.RecordsResponse{Records: []hetzner.Record{}}) // No existing record
			})
			mockAPI.SetHandler("POST", "/records", func(w http.ResponseWriter, r *http.Request) {
				var createReq hetzner.RecordRequest
				err := json.NewDecoder(r.Body).Decode(&createReq)
				Expect(err).NotTo(HaveOccurred())
				Expect(createReq.ZoneID).To(Equal(zoneID))
				Expect(createReq.Name).To(Equal(reqData.Name))
				// TTL check is implicitly handled by expecting the whole RecordRequest to match if needed
				writeJSONResponse(w, http.StatusCreated, hetzner.RecordResponse{Record: hetzner.Record{
					ID: newRecordID, ZoneID: zoneID, Name: reqData.Name, Value: reqData.Value, Type: reqData.Type,
				}})
			})
			
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})

		It("should successfully create the record and call the next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusOK))
			Expect(nextHandlerCalled).To(BeTrue())
		})
	})

	Context("when ReqData is not in context", func() {
		BeforeEach(func() {
			req = httptest.NewRequest("POST", "/update", nil) // Default context, no ReqData
			finalHandler.ServeHTTP(rr, req)
		})
		It("should return InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

	Context("when Zone ID is not found via API", func() {
		BeforeEach(func() {
			reqData := newTestReqData("myrecord", "nonexistent.com", "1.2.3.4", "A")
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Query().Get("name")).To(Equal(reqData.Zone))
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{Zones: []hetzner.Zone{}}) // Empty list
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should return InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})
	
	// API Error Handling Contexts
	Context("when Hetzner API for GetZones fails", func() {
		BeforeEach(func() {
			reqData := newTestReqData("myrecord", "example.com", "1.2.3.4", "A")
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Hetzner API GetZones Error", http.StatusInternalServerError)
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should result in InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

	Context("when Hetzner API for GetRecords fails", func() {
		BeforeEach(func() {
			reqData := newTestReqData("myrecord", "example.com", "1.2.3.4", "A")
			zoneID := "zone123"
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{Zones: []hetzner.Zone{{ID: zoneID, Name: reqData.Zone}}})
			})
			mockAPI.SetHandler("GET", fmt.Sprintf("/records?zone_id=%s", zoneID), func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Hetzner API GetRecords Error", http.StatusInternalServerError)
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should result in InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

	Context("when Hetzner API for CreateRecord fails", func() {
		BeforeEach(func() {
			reqData := newTestReqData("newrecord", "example.com", "1.2.3.4", "A")
			zoneID := "zone123"
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{Zones: []hetzner.Zone{{ID: zoneID, Name: reqData.Zone}}})
			})
			mockAPI.SetHandler("GET", fmt.Sprintf("/records?zone_id=%s", zoneID), func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.RecordsResponse{Records: []hetzner.Record{}}) // Trigger create
			})
			mockAPI.SetHandler("POST", "/records", func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Hetzner API CreateRecord Error", http.StatusInternalServerError)
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should result in InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})
	
	Context("when Hetzner API for UpdateRecord fails", func() {
		BeforeEach(func() {
			reqData := newTestReqData("myrecord", "example.com", "1.2.3.4", "A")
			zoneID := "zone123"
			recordID := "rec456"
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.ZonesResponse{Zones: []hetzner.Zone{{ID: zoneID, Name: reqData.Zone}}})
			})
			mockAPI.SetHandler("GET", fmt.Sprintf("/records?zone_id=%s", zoneID), func(w http.ResponseWriter, r *http.Request) {
				writeJSONResponse(w, http.StatusOK, hetzner.RecordsResponse{Records: []hetzner.Record{
					{ID: recordID, ZoneID: zoneID, Name: reqData.Name, Value: "old.value", Type: reqData.Type},
				}}) // Trigger update
			})
			mockAPI.SetHandler("PUT", fmt.Sprintf("/records/%s", recordID), func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Hetzner API UpdateRecord Error", http.StatusInternalServerError)
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should result in InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

	Context("when Hetzner API for GetZones returns malformed JSON", func() {
		BeforeEach(func() {
			reqData := newTestReqData("myrecord", "example.com", "1.2.3.4", "A")
			mockAPI.SetHandler("GET", "/zones", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, errWrite := w.Write([]byte("this is not valid json"))
				Expect(errWrite).NotTo(HaveOccurred())
			})
			ctx := middleware.NewContextWithReqData(context.Background(), reqData)
			req = httptest.NewRequest("POST", "/update", nil).WithContext(ctx)
			finalHandler.ServeHTTP(rr, req)
		})
		It("should result in InternalServerError and not call next handler", func() {
			Expect(rr.Code).To(Equal(http.StatusInternalServerError))
			Expect(nextHandlerCalled).To(BeFalse())
		})
	})

})
