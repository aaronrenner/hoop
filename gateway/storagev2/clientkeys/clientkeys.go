package clientkeysstorage

import (
	"crypto/sha256"
	"fmt"

	"github.com/google/uuid"
	"github.com/runopsio/hoop/gateway/storagev2"
	"github.com/runopsio/hoop/gateway/storagev2/types"
	"olympos.io/encoding/edn"
)

func GetEntity(ctx *storagev2.Context, xtID string) (*types.ClientKey, error) {
	data, err := ctx.GetEntity(xtID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var obj types.ClientKey
	return &obj, edn.Unmarshal(data, &obj)
}

func GetByName(ctx *storagev2.Context, name string) (*types.ClientKey, error) {
	payload := fmt.Sprintf(`{:query {
		:find [(pull ?c [*])] 
		:in [org name]
		:where [[?c :clientkey/org org]
				[?c :clientkey/name name]]}
		:in-args [%q %q]}`, ctx.OrgID, name)
	b, err := ctx.Query(payload)
	if err != nil {
		return nil, err
	}

	var clientKey [][]types.ClientKey
	if err := edn.Unmarshal(b, &clientKey); err != nil {
		return nil, err
	}

	if len(clientKey) == 0 {
		return nil, nil
	}

	return &clientKey[0][0], nil
}

func List(ctx *storagev2.Context) ([]types.ClientKey, error) {
	payload := fmt.Sprintf(`{:query {
		:find [(pull ?c [*])] 
		:in [org]
		:where [[?c :clientkey/org org]]}
		:in-args [%q]}`, ctx.OrgID)
	b, err := ctx.Query(payload)
	if err != nil {
		return nil, err
	}

	var clientKeyItems [][]types.ClientKey
	if err := edn.Unmarshal(b, &clientKeyItems); err != nil {
		return nil, err
	}

	var itemList []types.ClientKey
	for _, clientKey := range clientKeyItems {
		itemList = append(itemList, clientKey[0])
	}

	return itemList, nil
}

func ValidateDSN(store *storagev2.Store, dsn string) (*types.ClientKey, error) {
	dsnHash, err := sha256Hash(dsn)
	if err != nil {
		return nil, err
	}
	payload := fmt.Sprintf(`{:query {
		:find [(pull ?c [*])] 
		:in [dsnhash]
		:where [[?c :clientkey/enabled true]
				[?c :clientkey/dsnhash dsnhash]]}
		:in-args [%q]}`, dsnHash)
	b, err := store.Query(payload)
	if err != nil {
		return nil, err
	}

	var clientKey [][]types.ClientKey
	if err := edn.Unmarshal(b, &clientKey); err != nil {
		return nil, err
	}

	if len(clientKey) == 0 {
		return nil, nil
	}

	return &clientKey[0][0], nil

}

func Put(ctx *storagev2.Context, name string, active bool) (*types.ClientKey, string, error) {
	clientkey, err := GetByName(ctx, name)
	if err != nil {
		return nil, "", err
	}
	if clientkey == nil {
		keyHash, err := sha256Hash(uuid.NewString())
		if err != nil {
			return nil, "", err
		}
		dsn := fmt.Sprintf("%s/%s", ctx.ApiURL, keyHash)
		dsnHash, err := sha256Hash(dsn)
		if err != nil {
			return nil, "", err
		}
		obj := &types.ClientKey{
			ID:      uuid.NewString(),
			OrgID:   ctx.OrgID,
			Name:    name,
			DSNHash: dsnHash,
			Active:  active,
		}
		_, err = ctx.Put(obj)
		return obj, dsn, err
	}
	clientkey.Active = active
	_, err = ctx.Put(clientkey)
	return clientkey, "", err
}

func sha256Hash(data string) (string, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(data)); err != nil {
		return "", fmt.Errorf("failed generating hash, err=%v", err)
	}
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs), nil
}
