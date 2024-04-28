package backup

import (
	"context"
	"errors"
	"strings"

	"bytetrade.io/web3os/tapr/cmd/sys-event/watchers"
	aprv1 "bytetrade.io/web3os/tapr/pkg/apis/apr/v1alpha1"
	"bytetrade.io/web3os/tapr/pkg/backup"
	aprclientset "bytetrade.io/web3os/tapr/pkg/generated/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog/v2"
)

type Subscriber struct {
	*watchers.Subscriber
	aprClient     *aprclientset.Clientset
	dynamicClient *dynamic.DynamicClient
	invoker       *watchers.CallbackInvoker
}

func (s *Subscriber) WithKubeConfig(config *rest.Config) *Subscriber {
	s.aprClient = aprclientset.NewForConfigOrDie(config)
	s.dynamicClient = dynamic.NewForConfigOrDie(config)

	s.invoker = &watchers.CallbackInvoker{
		AprClient: s.aprClient,
		Retriable: func(err error) bool { return !strings.HasSuffix(err.Error(), errCancel.Error()) },
	}
	return s
}

func (s *Subscriber) HandleEvent() cache.ResourceEventHandler {
	return cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			return true
		},

		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				b := obj.(*backup.Backup)
				if watchers.ValidWatchDuration(&b.ObjectMeta) {
					eobj := watchers.EnqueueObj{
						Subscribe: s,
						Obj:       obj,
						Action:    watchers.ADD,
					}
					s.Watchers.Enqueue(eobj)
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				eobj := watchers.EnqueueObj{
					Subscribe: s,
					Obj:       newObj,
					Action:    watchers.UPDATE,
				}
				s.Watchers.Enqueue(eobj)
			},
		},
	}
}

func (s *Subscriber) Do(ctx context.Context, obj interface{}, action watchers.Action) error {
	b := obj.(*backup.Backup)

	_, phase, _ := backupStatus(b)

	postBackupInfo := &struct {
		Status string `json:"status"`
	}{
		Status: phase,
	}

	switch action {
	case watchers.ADD:
		if *b.Spec.Phase == backup.BackupNew {
			klog.Info("a new backup request received, send the event to the world")
			err := s.invoker.Invoke(ctx,
				func(cb *aprv1.SysEventRegistry) bool {
					return cb.Spec.Type == aprv1.Subscriber && cb.Spec.Event == aprv1.BackupNew
				},
				postBackupInfo,
			)

			if err != nil {
				klog.Error("send backup event error, ", err)
				return s.updateBackupPhase(ctx, b, backup.BackupCancel, err.Error())
			} else {
				klog.Info("success to send event, start backup")
				return s.updateBackupPhase(ctx, b, backup.BackupStart, "")
			}
		}
	case watchers.UPDATE:
		invoke := func(data interface{}) error {
			klog.Info("backup finished, send the event of the backup result to the world")
			return s.invoker.Invoke(ctx,
				func(cb *aprv1.SysEventRegistry) bool {
					return cb.Spec.Type == aprv1.Subscriber && cb.Spec.Event == aprv1.BackupFinish
				},
				data,
			)

		}

		switch phase {
		case backup.BackupFailed, backup.BackupSucceed, backup.BackupCancel:
			return invoke(postBackupInfo)
		}
	}

	return nil
}

var errCancel error = errors.New("forbidden, canceled")

func (s *Subscriber) updateBackupPhase(ctx context.Context, b *backup.Backup, phase string, errMsg string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		data, err := s.dynamicClient.Resource(backup.BackupGVR).Namespace(b.Namespace).Get(ctx, b.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		var updateBackup backup.Backup
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(data.Object, &updateBackup)
		if err != nil {
			klog.Error("convert unstructured error, ", err)
			return err
		}

		// FIXME: user patch instead
		updateBackup.Spec.Phase = &phase
		updateBackup.Spec.FailedMessage = &errMsg

		updateData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&updateBackup)
		if err != nil {
			klog.Error("convert to unstructured error, ", err, ", ", updateBackup.Name, ", ", updateBackup.Namespace)
			return err
		}

		_, err = s.dynamicClient.Resource(backup.BackupGVR).Namespace(b.Namespace).Update(ctx,
			&unstructured.Unstructured{Object: updateData}, metav1.UpdateOptions{})
		return err
	})

}

func backupStatus(sb *backup.Backup) (bool, string, error) {
	var (
		phase *string
		err   error
	)

	phase = sb.Spec.ResticPhase
	if phase == nil {
		switch {
		case sb.Spec.Phase != nil:
			switch *sb.Spec.Phase {
			case backup.BackupNew, backup.BackupCancel:
				return false, *sb.Spec.Phase, nil
			case backup.FinalizingPartiallyFailed,
				backup.PartiallyFailed,
				backup.BackupFailed,
				backup.FailedValidation:
				return false, backup.BackupFailed, errors.New(*sb.Spec.FailedMessage)
			case backup.VeleroBackupCompleted:
				if sb.Spec.MiddleWarePhase == nil {
					return false, backup.BackupRunning, nil
				}

				switch *sb.Spec.MiddleWarePhase {
				case backup.BackupRunning:
					return false, backup.BackupRunning, nil
				case backup.BackupFailed:
					err = errors.New("middleware backup failed")
					if sb.Spec.MiddleWareFailedMessage != nil {
						err = errors.New(*sb.Spec.MiddleWareFailedMessage)
					}
					return false, backup.BackupFailed, err
				}

			}
		}

		return false, "", errors.New("unknown backup status")
	}

	switch *phase {
	case backup.BackupSucceed:
		return true, *phase, nil
	case backup.BackupRunning:
		return false, *phase, nil
	case backup.BackupFailed:
		return false, *phase, errors.New(*sb.Spec.ResticFailedMessage)
	}

	return false, "", errors.New("unknown backup status")
}
