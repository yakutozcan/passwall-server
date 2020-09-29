package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/passwall/passwall-server/internal/app"
	"github.com/passwall/passwall-server/internal/storage"
	"github.com/passwall/passwall-server/model"

	"github.com/gorilla/mux"
)

const (
	SubscriptionDeleteSuccess = "Subscription deleted successfully!"
)

func getAlertName(r *http.Request) (string, error) {
	alertNameDTO := new(model.AlertName)

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&alertNameDTO); err != nil {
		return "", err
	}
	// defer r.Body.Close()

	return alertNameDTO.AlertName, nil
}

// PostSubscription ...
func PostSubscription(s storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		bodyBytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
			return
		}

		bodyMap := make(map[string]string)
		err = json.Unmarshal(bodyBytes, &bodyMap)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
			return
		}

		r.Body.Close() //  must close

		// Generate body again for alert type
		r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

		if bodyMap["alert_name"] == "subscription_created" {
			subscriptionCreated := new(model.SubscriptionCreated)

			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&subscriptionCreated); err != nil {
				RespondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
				return
			}
			defer r.Body.Close()

			schema := "public"
			subscriptionResult, err := s.Subscriptions().Save(model.FromCreToSub(subscriptionCreated), schema)
			if err != nil {
				RespondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		//subscriptionDTO := new(AlertName)

		// TODO: There are 6 action here. These should be moved to service layer
		// user's service layer functions located in /app/user.go file is

		// 1. Decode request body to userDTO object
		// decoder := json.NewDecoder(r.Body)
		// if err := decoder.Decode(&subscriptionDTO); err != nil {
		// 	RespondWithError(w, http.StatusBadRequest, "Invalid resquest payload")
		// 	return
		// }
		// defer r.Body.Close()

		// j, _ := json.Marshal(subscriptionDTO)
		// fmt.Println(string(j))

		// schema := "public"
		// createdSubscription, err := s.Subscriptions().Save(model.ToSubscription(&subscriptionDTO), schema)
		// if err != nil {
		// 	RespondWithError(w, http.StatusInternalServerError, err.Error())
		// 	return
		// }

		// createdSubscriptionDTO := model.ToSubscriptionDTO(createdSubscription)

		RespondWithJSON(w, http.StatusOK, subscriptionResult)
	}
}

// FindAllSubscriptions ...
func FindAllSubscriptions(s storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error
		var subscriptionList []model.Subscription

		fields := []string{"id", "created_at", "updated_at", "title", "ip", "url"}
		argsStr, argsInt := SetArgs(r, fields)

		schema := "public"
		subscriptionList, err = s.Subscriptions().FindAll(argsStr, argsInt, schema)

		if err != nil {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		// Encrypt payload
		var payload model.Payload
		key := r.Context().Value("transmissionKey").(string)
		encrypted, err := app.EncryptJSON(key, subscriptionList)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload.Data = string(encrypted)

		RespondWithJSON(w, http.StatusOK, payload)
	}
}

// FindSubscriptionByID ...
func FindSubscriptionByID(s storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		schema := "public"
		subscription, err := s.Subscriptions().FindByID(uint(id), schema)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		// Decrypt subscription side encrypted fields
		decSubscription, err := app.DecryptModel(subscription)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		subscriptionDTO := model.ToSubscriptionDTO(decSubscription.(*model.Subscription))

		// Encrypt payload
		var payload model.Payload
		key := r.Context().Value("transmissionKey").(string)
		encrypted, err := app.EncryptJSON(key, subscriptionDTO)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload.Data = string(encrypted)

		RespondWithJSON(w, http.StatusOK, payload)
	}
}

// CreateSubscription ...
func CreateSubscription(s storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		payload, err := ToPayload(r)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, InvalidRequestPayload)
			return
		}
		defer r.Body.Close()

		// Decrypt payload
		var subscriptionDTO model.SubscriptionDTO
		key := r.Context().Value("transmissionKey").(string)
		err = app.DecryptJSON(key, []byte(payload.Data), &subscriptionDTO)
		if err != nil {
			fmt.Println("burada")
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		schema := "public"
		createdSubscription, err := s.Subscriptions().Save(model.ToSubscription(&subscriptionDTO), schema)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		createdSubscriptionDTO := model.ToSubscriptionDTO(createdSubscription)

		// Encrypt payload
		encrypted, err := app.EncryptJSON(key, createdSubscriptionDTO)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload.Data = string(encrypted)

		RespondWithJSON(w, http.StatusOK, payload)
	}
}

// UpdateSubscription ...
// func UpdateSubscription(s storage.Store) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		vars := mux.Vars(r)
// 		id, err := strconv.Atoi(vars["id"])
// 		if err != nil {
// 			RespondWithError(w, http.StatusBadRequest, err.Error())
// 			return
// 		}

// 		// Unmarshal request body to payload
// 		var payload model.Payload
// 		decoder := json.NewDecoder(r.Body)
// 		if err := decoder.Decode(&payload); err != nil {
// 			RespondWithError(w, http.StatusBadRequest, InvalidRequestPayload)
// 			return
// 		}
// 		defer r.Body.Close()

// 		// Decrypt payload
// 		var subscriptionDTO model.SubscriptionDTO
// 		key := r.Context().Value("transmissionKey").(string)
// 		err = app.DecryptJSON(key, []byte(payload.Data), &subscriptionDTO)
// 		if err != nil {
// 			RespondWithError(w, http.StatusInternalServerError, err.Error())
// 			return
// 		}

// 		schema := "public"
// 		subscription, err := s.Subscriptions().FindByID(uint(id), schema)
// 		if err != nil {
// 			RespondWithError(w, http.StatusNotFound, err.Error())
// 			return
// 		}

// 		updatedSubscription, err := s.Subscriptions().Save(model.ToSubscription(&subscriptionDTO), schema) app.UpdateSubscription(s, subscription, &subscriptionDTO, schema)
// 		if err != nil {
// 			RespondWithError(w, http.StatusInternalServerError, err.Error())
// 			return
// 		}

// 		updatedSubscriptionDTO := model.ToSubscriptionDTO(updatedSubscription)

// 		// Encrypt payload
// 		encrypted, err := app.EncryptJSON(key, updatedSubscriptionDTO)
// 		if err != nil {
// 			RespondWithError(w, http.StatusInternalServerError, err.Error())
// 			return
// 		}
// 		payload.Data = string(encrypted)

// 		RespondWithJSON(w, http.StatusOK, payload)
// 	}
// }

// DeleteSubscription ...
func DeleteSubscription(s storage.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, err := strconv.Atoi(vars["id"])
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, err.Error())
			return
		}

		schema := "public"
		subscription, err := s.Subscriptions().FindByID(uint(id), schema)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		err = s.Subscriptions().Delete(subscription.ID, schema)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		response := model.Response{
			Code:    http.StatusOK,
			Status:  Success,
			Message: SubscriptionDeleteSuccess,
		}
		RespondWithJSON(w, http.StatusOK, response)
	}
}