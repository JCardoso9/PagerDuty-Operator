package k8s_utils

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func DoNotRequeue() (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func Requeue() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

func RequeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}
