package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
	apiserverserviceaccount "k8s.io/apiserver/pkg/authentication/serviceaccount"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/serviceaccount"
)

// time.Now stubbed out to allow testing
var now = time.Now

type privateClaims struct {
	Kubernetes kubernetes `json:"kubernetes.io,omitempty"`
}

type ref struct {
	Name string `json:"name,omitempty"`
	UID  string `json:"uid,omitempty"`
}

type kubernetes struct {
	Namespace string          `json:"namespace,omitempty"`
	Svcacct   ref             `json:"serviceaccount,omitempty"`
	Pod       *ref            `json:"pod,omitempty"`
	Secret    *ref            `json:"secret,omitempty"`
	WarnAfter jwt.NumericDate `json:"warnafter,omitempty"`
}

type validator struct {
	getter serviceaccount.ServiceAccountTokenGetter
}

var _ = serviceaccount.Validator(&validator{})

func NewValidator(getter serviceaccount.ServiceAccountTokenGetter) serviceaccount.Validator {
	return &validator{
		getter: getter,
	}
}

func (v *validator) Validate(_ context.Context, _ string, public *jwt.Claims, privateObj interface{}) (*apiserverserviceaccount.ServiceAccountInfo, error) {
	private, ok := privateObj.(*privateClaims)
	if !ok {
		klog.Errorf("service account jwt validator expected private claim of type *privateClaims but got: %T", privateObj)
		return nil, fmt.Errorf("service account token claims could not be validated due to unexpected private claim")
	}
	nowTime := now()
	err := public.Validate(jwt.Expected{
		Time: nowTime,
	})
	switch err {
	case nil:
		// successful validation

	case jwt.ErrExpired:
		return nil, fmt.Errorf("service account token has expired")

	case jwt.ErrNotValidYet:
		return nil, fmt.Errorf("service account token is not valid yet")

	// our current use of jwt.Expected above should make these cases impossible to hit
	case jwt.ErrInvalidAudience, jwt.ErrInvalidID, jwt.ErrInvalidIssuer, jwt.ErrInvalidSubject:
		klog.Errorf("service account token claim validation got unexpected validation failure: %v", err)
		return nil, fmt.Errorf("service account token claims could not be validated: %w", err) // safe to pass these errors back to the user

	default:
		klog.Errorf("service account token claim validation got unexpected error type: %T", err)                         // avoid leaking unexpected information into the logs
		return nil, fmt.Errorf("service account token claims could not be validated due to unexpected validation error") // return an opaque error
	}

	// consider things deleted prior to now()-leeway to be invalid
	invalidIfDeletedBefore := nowTime.Add(-jwt.DefaultLeeway)
	namespace := private.Kubernetes.Namespace
	saref := private.Kubernetes.Svcacct
	podref := private.Kubernetes.Pod
	secref := private.Kubernetes.Secret
	// Make sure service account still exists (name and UID)
	serviceAccount, err := v.getter.GetServiceAccount(namespace, saref.Name)
	if err != nil {
		klog.V(4).Infof("Could not retrieve service account %s/%s: %v", namespace, saref.Name, err)
		return nil, err
	}
	if serviceAccount.DeletionTimestamp != nil && serviceAccount.DeletionTimestamp.Time.Before(invalidIfDeletedBefore) {
		klog.V(4).Infof("Service account has been deleted %s/%s", namespace, saref.Name)
		return nil, fmt.Errorf("service account %s/%s has been deleted", namespace, saref.Name)
	}
	if string(serviceAccount.UID) != saref.UID {
		klog.V(4).Infof("Service account UID no longer matches %s/%s: %q != %q", namespace, saref.Name, string(serviceAccount.UID), saref.UID)
		return nil, fmt.Errorf("service account UID (%s) does not match claim (%s)", serviceAccount.UID, saref.UID)
	}

	if secref != nil {
		// Make sure token hasn't been invalidated by deletion of the secret
		secret, err := v.getter.GetSecret(namespace, secref.Name)
		if err != nil {
			klog.V(4).Infof("Could not retrieve bound secret %s/%s for service account %s/%s: %v", namespace, secref.Name, namespace, saref.Name, err)
			return nil, fmt.Errorf("service account token has been invalidated")
		}
		if secret.DeletionTimestamp != nil && secret.DeletionTimestamp.Time.Before(invalidIfDeletedBefore) {
			klog.V(4).Infof("Bound secret is deleted and awaiting removal: %s/%s for service account %s/%s", namespace, secref.Name, namespace, saref.Name)
			return nil, fmt.Errorf("service account token has been invalidated")
		}
		if secref.UID != string(secret.UID) {
			klog.V(4).Infof("Secret UID no longer matches %s/%s: %q != %q", namespace, secref.Name, string(secret.UID), secref.UID)
			return nil, fmt.Errorf("secret UID (%s) does not match service account secret ref claim (%s)", secret.UID, secref.UID)
		}
	}

	var podName, podUID string
	if podref != nil {
		// Make sure token hasn't been invalidated by deletion of the pod
		pod, err := v.getter.GetPod(namespace, podref.Name)
		if err != nil {
			klog.V(4).Infof("Could not retrieve bound pod %s/%s for service account %s/%s: %v", namespace, podref.Name, namespace, saref.Name, err)
			return nil, fmt.Errorf("service account token has been invalidated")
		}
		if pod.DeletionTimestamp != nil && pod.DeletionTimestamp.Time.Before(invalidIfDeletedBefore) {
			klog.V(4).Infof("Bound pod is deleted and awaiting removal: %s/%s for service account %s/%s", namespace, podref.Name, namespace, saref.Name)
			return nil, fmt.Errorf("service account token has been invalidated")
		}
		if podref.UID != string(pod.UID) {
			klog.V(4).Infof("Pod UID no longer matches %s/%s: %q != %q", namespace, podref.Name, string(pod.UID), podref.UID)
			return nil, fmt.Errorf("pod UID (%s) does not match service account pod ref claim (%s)", pod.UID, podref.UID)
		}
		podName = podref.Name
		podUID = podref.UID
	}

	return &apiserverserviceaccount.ServiceAccountInfo{
		Namespace: private.Kubernetes.Namespace,
		Name:      private.Kubernetes.Svcacct.Name,
		UID:       private.Kubernetes.Svcacct.UID,
		PodName:   podName,
		PodUID:    podUID,
	}, nil
}

func (v *validator) NewPrivateClaims() interface{} {
	return &privateClaims{}
}

func parseSigned(tokenData string, dest ...interface{}) error {
	sig, err := jose.ParseSigned(tokenData)
	if err != nil {
		klog.Errorf("failed to parse token: %v", err)
		return err
	}
	claims := func(dest ...interface{}) error {
		b := sig.UnsafePayloadWithoutVerification()
		for _, d := range dest {
			if err := json.Unmarshal(b, d); err != nil {
				return err
			}
		}
		return nil
	}
	if err := claims(dest...); err != nil {
		klog.Errorf("failed to parse claims: %v", err)
		return err
	}
	return nil
}
