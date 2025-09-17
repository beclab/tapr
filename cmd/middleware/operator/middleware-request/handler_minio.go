package middlewarerequest

import (
	"context"
	"fmt"

	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	wminio "bytetrade.io/web3os/tapr/pkg/workload/minio"

	"github.com/minio/madmin-go"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"k8s.io/klog/v2"
)

func (c *controller) createOrUpdateMinioRequest(req *aprv1.MiddlewareRequest) error {
	adminUser, adminPassword, err := c.findMinioAdminCredentials(req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to find minio admin credentials: %w", err)
	}

	endpoint, err := c.getMinioEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get minio endpoint: %w", err)
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(adminUser, adminPassword, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	madminClient, err := madmin.New(endpoint, adminUser, adminPassword, false)
	if err != nil {
		return fmt.Errorf("failed to create minio admin client: %v", err)
	}

	klog.Info("create minio user and buckets, ", req.Spec.Minio.User)

	userPassword, err := req.Spec.Minio.Password.GetVarValue(c.ctx, c.k8sClientSet, req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get user password: %w", err)
	}

	err = c.createOrUpdateMinioUser(c.ctx, madminClient, req.Spec.Minio.User, userPassword)
	if err != nil {
		return fmt.Errorf("failed to create or update minio user: %w", err)
	}

	for _, bucket := range req.Spec.Minio.Buckets {
		bucketName := c.getMinioRealBucketName(req.Spec.AppNamespace, bucket.Name)
		klog.Info("create bucket for user, ", bucketName, ", ", req.Spec.Minio.User)

		err = minioClient.MakeBucket(c.ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			exists, errBucketExists := minioClient.BucketExists(c.ctx, bucketName)
			if errBucketExists != nil {
				return fmt.Errorf("failed to check bucket existence: %w", errBucketExists)
			}
			if !exists {
				return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
			}
			klog.Info("bucket already exists, ", bucketName)
		}

		err = c.setBucketPolicyForUser(c.ctx, madminClient, bucketName, req.Spec.Minio.User)
		if err != nil {
			return fmt.Errorf("failed to set bucket policy: %w", err)
		}

	}

	return nil
}

func (c *controller) deleteMinioRequest(req *aprv1.MiddlewareRequest) error {
	adminUser, adminPassword, err := c.findMinioAdminCredentials(req.Namespace)
	if err != nil {
		return fmt.Errorf("failed to find minio admin credentials: %w", err)
	}

	endpoint, err := c.getMinioEndpoint()
	if err != nil {
		return fmt.Errorf("failed to get minio endpoint: %w", err)
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(adminUser, adminPassword, ""),
		Secure: false,
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}
	madminClient, err := madmin.New(endpoint, adminUser, adminPassword, false)
	if err != nil {
		return fmt.Errorf("failed to create minio admin client: %v", err)
	}

	klog.Info("delete minio user and buckets, ", req.Spec.Minio.User)

	for _, bucket := range req.Spec.Minio.Buckets {
		bucketName := c.getMinioRealBucketName(req.Spec.AppNamespace, bucket.Name)
		klog.Info("delete bucket, ", bucketName)

		err = c.removeAllObjectsInBucket(c.ctx, minioClient, bucketName)
		if err != nil {
			klog.Warning("failed to remove objects in bucket ", bucketName, ": ", err)
		}

		err = minioClient.RemoveBucket(c.ctx, bucketName)
		if err != nil {
			klog.Warning("failed to remove bucket ", bucketName, ": ", err)
		}
	}

	err = c.deleteMinioUser(c.ctx, madminClient, req.Spec.Minio.User)
	if err != nil {
		return fmt.Errorf("failed to delete minio user: %w", err)
	}

	return nil
}

func (c *controller) findMinioAdminCredentials(namespace string) (string, string, error) {
	return wminio.FindMinioAdminUser(c.ctx, c.k8sClientSet, "minio-middleware")
}

func (c *controller) getMinioEndpoint() (string, error) {
	return fmt.Sprintf("minio-minio.%s.svc.cluster.local:9000", "minio-middleware"), nil
}

func (c *controller) getMinioRealBucketName(appNamespace, bucketName string) string {
	return fmt.Sprintf("%s-%s", appNamespace, bucketName)
}

func (c *controller) createOrUpdateMinioUser(ctx context.Context, madminClient *madmin.AdminClient, username, password string) error {

	err := madminClient.AddUser(ctx, username, password)
	if err != nil {
		return fmt.Errorf("failed to add/update user: %v", err)
	}
	klog.Info("creating or updating minio user: ", username)
	return nil
}

func (c *controller) setBucketPolicyForUser(ctx context.Context, madminClient *madmin.AdminClient, bucketName, username string) error {
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Action": "s3:*",
				"Resource": [
					"arn:aws:s3:::%s",
					"arn:aws:s3:::%s/*"
				]
			}
		]
	}`, bucketName, bucketName)
	policyName := fmt.Sprintf("%s-policy", username)
	err := madminClient.AddCannedPolicy(ctx, policyName, []byte(policy))
	if err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	err = madminClient.SetPolicy(ctx, policyName, username, false)
	if err != nil {
		return fmt.Errorf("failed to set policy: %s for user: %s, err %v", policyName, username, err)
	}
	klog.Infof("set bucket policy for user %s on bucket %s", username, bucketName)
	return nil
}

func (c *controller) removeAllObjectsInBucket(ctx context.Context, client *minio.Client, bucketName string) error {
	objectsCh := client.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true})
	for object := range objectsCh {
		if object.Err != nil {
			return object.Err
		}
		err := client.RemoveObject(ctx, bucketName, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			klog.Warning("failed to remove object ", object.Key, " from bucket ", bucketName, ": ", err)
		}
	}
	return nil
}

func (c *controller) deleteMinioUser(ctx context.Context, madminClient *madmin.AdminClient, username string) error {
	users, err := madminClient.ListUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list users: %v", err)
	}
	if _, exists := users[username]; !exists {
		klog.Infof("User %s does not exist, skipping deletion", username)
		return nil
	}

	err = madminClient.RemoveUser(ctx, username)
	if err != nil {
		return fmt.Errorf("failed to remove user %s: %v", username, err)
	}
	klog.Infof("Deleted minio user: %s", username)
	return nil
}
