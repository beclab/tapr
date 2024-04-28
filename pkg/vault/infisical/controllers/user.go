package controllers

import (
	"errors"
	"net/http"

	"bytetrade.io/web3os/tapr/pkg/vault/infisical"
	"k8s.io/klog/v2"
)

type userClient struct {
}

func (u *userClient) GetUserOrganizationId(token string) (string, error) {

	url := infisical.InfisicalAddr + "/api/v2/users/me/organizations"

	client := NewHttpClient()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+token).
		SetResult(&Organizations{}).
		Get(url)

	if err != nil {
		klog.Error("get user organiztions error, ", err)
		return "", err
	}

	if resp.StatusCode() != http.StatusOK {
		klog.Error("get user organiztions error, ", string(resp.Body()))
		return "", errors.New(string(resp.Body()))
	}

	orgs := resp.Result().(*Organizations)
	if len(orgs.Items) == 0 {
		return "", errors.New("user doesn't has organizations")
	}

	return orgs.Items[0].Id, nil
}

func (u *userClient) GetUserPrivateKey(user *infisical.User, password string) (string, error) {
	return infisical.DecryptUserPrivateKeyHelper(user, password)
}
