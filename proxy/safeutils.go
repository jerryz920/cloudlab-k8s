package main

import (
	"encoding/json"
	"errors"
	"net/http"

	logrus "github.com/sirupsen/logrus"
)

/// This function is no longer used
func GetPrincipalID(resp *http.Response) (string, error) {
	if resp.StatusCode != http.StatusOK {
		logrus.Debugf("error getting response %v\n", resp.StatusCode)
		return "", errors.New("error state " + resp.Status)
	}
	decoder := json.NewDecoder(resp.Body)
	pr := PrincipalResponse{}

	if err := decoder.Decode(&pr); err != nil {
		logrus.Debugf("error in decoding %v\n", err)
		return "", err
	}
	/// message is in ['<ID>'] form
	var matches []string
	if matches = pidMatch.FindStringSubmatch(pr.Message); len(matches) != 2 {
		logrus.Debugf("error finding PID: %v", pr.Message)
		return "", errors.New("error finding pid in response")
	}
	return matches[1], nil
}
