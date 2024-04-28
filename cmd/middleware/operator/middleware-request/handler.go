package middlewarerequest

import (
	"errors"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
)

func (c *controller) handler(action Action, obj interface{}) error {
	request, ok := obj.(*aprv1.MiddlewareRequest)
	if !ok {
		return errors.New("invalid object")
	}

	switch request.Spec.Middleware {
	case aprv1.TypePostgreSQL:
		switch action {
		case ADD, UPDATE:
			// create app db user
			err := c.createOrUpdatePGRequest(request)
			if err != nil {
				return err
			}

			if action == UPDATE {
				// delete db if not in request
				err = c.deleteDatabaseIfNotExists(request)
				if err != nil {
					return err
				}
			}

		case DELETE:
			err := c.deletePGAll(request)
			if err != nil {
				return err
			}
		}
	case aprv1.TypeMongoDB:
		switch action {
		case ADD, UPDATE:
			if err := c.createOrUpdateMDBRequest(request); err != nil {
				return err
			}

		case DELETE:
			if err := c.deleteMDBRequest(request); err != nil {
				return err
			}
		}
	case aprv1.TypeRedis:
		if err := c.reconcileRedisPassword(request); err != nil {
			return err
		}
	case aprv1.TypeZinc:
		switch action {
		case ADD, UPDATE:
			if err := c.createOrUpdataIndexForUser(request); err != nil {
				return err
			}

		case DELETE:
			if err := c.deleteIndexAndUser(request); err != nil {
				return err
			}
		}
	}

	return nil
}
