package controllers

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// RestartCountThreshold is the number of restarts after which we consider a pod failing.
	RestartCountThreshold = 3
)

// PodFailureReasons we detect.
const (
	ReasonCrashLoopBackOff  = "CrashLoopBackOff"
	ReasonImagePullBackOff  = "ImagePullBackOff"
	ReasonErrImagePull      = "ErrImagePull"
	ReasonOOMKilled         = "OOMKilled"
	ReasonError             = "Error"
	ReasonBackOffRestarts   = "BackOffRestarts"
)

// IsPodFailed returns true if the pod shows failure signals, and a short reason.
func IsPodFailed(pod *corev1.Pod) (bool, string) {
	if pod == nil {
		return false, ""
	}

	// Phase
	if pod.Status.Phase == corev1.PodFailed {
		return true, string(corev1.PodFailed)
	}

	// Container statuses
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil {
			r := cs.State.Waiting.Reason
			switch r {
			case ReasonCrashLoopBackOff, ReasonImagePullBackOff, ReasonErrImagePull:
				return true, r
			}
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason == ReasonOOMKilled {
			return true, ReasonOOMKilled
		}
		if cs.RestartCount >= RestartCountThreshold {
			return true, ReasonBackOffRestarts
		}
	}

	return false, ""
}

// IsJobFailed returns true if the job has failed or completed with failure.
func IsJobFailed(job *batchv1.Job) (bool, string) {
	if job == nil {
		return false, ""
	}
	if job.Status.Failed > 0 {
		return true, "JobFailed"
	}
	if job.Status.Succeeded > 0 && job.Status.Failed > 0 {
		return true, "JobCompletedWithFailures"
	}
	// Completed but with failure condition
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true, "JobFailed"
		}
	}
	return false, ""
}

// IsDeploymentFailed infers failure from deployment status. We consider it failed
// if unavailable replicas exist and we're past the progress deadline (or simple: unavailable > 0).
// Caller may also resolve failure by checking owned Pods.
func IsDeploymentFailed(dep *appsv1.Deployment) (bool, string) {
	if dep == nil {
		return false, ""
	}
	if dep.Status.UnavailableReplicas > 0 {
		return true, "UnavailableReplicas"
	}
	for _, c := range dep.Status.Conditions {
		if c.Type == appsv1.DeploymentReplicaFailure && c.Status == corev1.ConditionTrue {
			return true, "DeploymentReplicaFailure"
		}
	}
	return false, ""
}
