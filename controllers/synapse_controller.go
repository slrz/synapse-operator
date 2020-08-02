/*
Copyright © 2020 The synapse-operator Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	matrixv1alpha1 "github.com/slrz/synapse-operator/api/v1alpha1"
	"github.com/slrz/synapse-operator/pkg/synapseconf"
)

// SynapseReconciler reconciles a Synapse object
type SynapseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=matrix.slrz.net,resources=synapsis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=matrix.slrz.net,resources=synapsis/status,verbs=get;update;patch

func (r *SynapseReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("synapse", req.NamespacedName)

	synapse := &matrixv1alpha1.Synapse{}
	err := r.Get(ctx, req.NamespacedName, synapse)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("get Synapse: not found, ignoring")
			return ctrl.Result{}, nil
		}
		// requeue request
		log.Error(err, "get Synapse")
		return ctrl.Result{}, err
	}

	// Create secret if it doesn't exist yet
	secret := &v1.Secret{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      synapse.Name,
		Namespace: synapse.Namespace,
	}, secret)
	if err != nil && errors.IsNotFound(err) {
		// Need to create it
		secret := synapseSecret(synapse)
		ctrl.SetControllerReference(synapse, secret, r.Scheme)
		log.Info("creating Secret",
			"Secret.Namespace", secret.Namespace,
			"Secret.Name", secret.Name)
		err = r.Create(ctx, secret)
		if err != nil {
			log.Error(err, "create Secret",
				"Secret.Namespace", secret.Namespace,
				"Secret.Name", secret.Name)
			return ctrl.Result{}, err
		}
		// Secret created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		log.Error(err, "get Secret")
		return ctrl.Result{}, err
	}

	// Ensure the config map exists…
	cm := &v1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      synapse.Name,
		Namespace: synapse.Namespace,
	}, cm)
	if err != nil && errors.IsNotFound(err) {
		cm := synapseConfigMap(synapse, secret)
		ctrl.SetControllerReference(synapse, cm, r.Scheme)
		log.Info("creating ConfigMap",
			"ConfigMap.Namespace", cm.Namespace,
			"ConfigMap.Name", cm.Name)
		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "create ConfigMap",
				"ConfigMap.Namespace", cm.Namespace,
				"ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}
	if err != nil {
		log.Error(err, "get ConfigMap")
		return ctrl.Result{}, err
	}

	// … and is still in sync with the CR spec.
	config, wantDigest := homeserverConfigFromCR(synapse, secret)
	if gotDigest := cm.Annotations[inputIDAnnotationKey]; wantDigest != gotDigest {
		log.Info("ConfigMap needs update",
			"ConfigMap.Namespace", cm.Namespace,
			"ConfigMap.Name", cm.Name,
			"wantDigest", wantDigest, "gotDigest", gotDigest)
		yamlBytes, err := synapseconf.GenerateHomeserverYAML(config)
		if err != nil {
			log.Error(err, "update ConfigMap: GenerateHomeserverYAML",
				"ConfigMap.Namespace", cm.Namespace,
				"ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		cm.Data["homeserver.yaml"] = string(yamlBytes)
		cm.Annotations[inputIDAnnotationKey] = wantDigest
		err = r.Update(ctx, cm)
		if err != nil {
			log.Error(err, "update ConfigMap",
				"ConfigMap.Namespace", cm.Namespace,
				"ConfigMap.Name", cm.Name)
			return ctrl.Result{}, err
		}
		// Updated CM - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

	// Now that the prerequisites exist, ensure we have a deployment
	dep := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      synapse.Name,
		Namespace: synapse.Namespace,
	}, dep)
	if err != nil && errors.IsNotFound(err) {
		dep := synapseDeployment(synapse, secret, cm)
		ctrl.SetControllerReference(synapse, dep, r.Scheme)
		log.Info("creating Deployment",
			"Deployment.Namespace", dep.Namespace,
			"Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "create Deployment",
				"Deployment.Namespace", dep.Namespace,
				"Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil

	}

	return ctrl.Result{}, nil
}

func (r *SynapseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&matrixv1alpha1.Synapse{}).
		Owns(&v1.Secret{}).
		Owns(&v1.ConfigMap{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func synapseSecret(cr *matrixv1alpha1.Synapse) *v1.Secret {
	var keyID string
	for {
		// We don't want '-' or '_' in the resulting keyID, so retry
		// until we get what we desire.
		keyID = randomString(4)
		if strings.IndexAny(keyID, "-_") == -1 {
			break
		}
	}
	signingKey, err := synapseconf.GenerateSigningKey(fmt.Sprintf("a_%s", keyID))
	if err != nil {
		// Signing key generation fails iff the system RNG is broken.
		// We can accept a panic in that case.
		panic(err)
	}
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    synapseLabels(cr.Name),
		},
		Data: map[string][]byte{
			"signing-key":                signingKey,
			"registration-shared-secret": []byte(randomString(64)),
			"macaroon-secret-key":        []byte(randomString(64)),
			"form-secret":                []byte(randomString(64)),
		},
		Type: "Opaque",
	}
}

const inputIDAnnotationKey = "matrix.slrz.net/input-identifier"

func synapseConfigMap(cr *matrixv1alpha1.Synapse, secret *v1.Secret) *v1.ConfigMap {
	// When attached to the config map, the digest allows us to detect when
	// the generated config file has become stale in relation to the inputs
	// it was generated from.
	config, dgst := homeserverConfigFromCR(cr, secret)

	yamlBytes, err := synapseconf.GenerateHomeserverYAML(config)
	if err != nil {
		// BUG(ls): Properly return an error instead of panicking.
		// Unlikely to be an issue until we support user-provided
		// templates for homeserver.yaml but still.
		panic(err)
	}

	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    synapseLabels(cr.Name),
			Annotations: map[string]string{
				inputIDAnnotationKey: dgst,
			},
		},
		Data: map[string]string{
			"homeserver.yaml":       string(yamlBytes),
			"homeserver.log.config": synapseLogConfig(),
		},
	}
}

func synapseDeployment(cr *matrixv1alpha1.Synapse, secret *v1.Secret, cm *v1.ConfigMap) *appsv1.Deployment {
	ls := synapseLabels(cr.Name)
	replicas := int32(1)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Volumes: synapseVolumes(secret, cm),
					Containers: []v1.Container{{
						Image: "docker.io/matrixdotorg/synapse:v1.18.0",
						Name:  "synapse",
						Ports: []v1.ContainerPort{{
							ContainerPort: 8008,
							Name:          "http",
						}},
						VolumeMounts: synapseVolumeMounts(cr),
					}},
				},
			},
		},
	}
}

func synapseVolumes(secret *v1.Secret, cm *v1.ConfigMap) []v1.Volume {
	return []v1.Volume{
		{
			Name: "data",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: "secrets",
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		},
		{
			Name: "config",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: cm.Name,
					},
				},
			},
		},
	}
}

func synapseVolumeMounts(cr *matrixv1alpha1.Synapse) []v1.VolumeMount {
	const (
		homeserverYAMLFilename = "homeserver.yaml"
		signingKeyFilename     = "homeserver.signing.key"
		logConfigFilename      = "homeserver.log.config"
	)

	return []v1.VolumeMount{
		{
			Name:      "data",
			MountPath: "/data",
		},
		{
			Name:      "config",
			MountPath: path.Join("/data", homeserverYAMLFilename),
			SubPath:   homeserverYAMLFilename,
			ReadOnly:  true,
		},
		{
			Name:      "config",
			MountPath: path.Join("/data", logConfigFilename),
			SubPath:   logConfigFilename,
			ReadOnly:  true,
		},
		{
			Name:      "secrets",
			MountPath: path.Join("/data", signingKeyFilename),
			SubPath:   signingKeyFilename,
			ReadOnly:  true,
		},
	}
}

func synapseLabels(name string) map[string]string {
	return map[string]string{"app": "synapse", "synapse_cr": name}
}

func homeserverConfigFromCR(cr *matrixv1alpha1.Synapse, secret *v1.Secret) (c *synapseconf.HomeserverConfig, id string) {
	config := &synapseconf.HomeserverConfig{
		ServerName:  cr.Spec.ServerName,
		ReportStats: cr.Spec.ReportStats,

		RegistrationSharedSecret: string(secret.Data["registration-shared-secret"]),
		MacaroonSecretKey:        string(secret.Data["macaroon-secret-key"]),
		FormSecret:               string(secret.Data["form-secret"]),
	}
	// Compute a digest over the inputs of homeserver.yaml generation.
	// Input variations change the digest and we can re-generate the
	// config.
	h := sha256.New()

	t := reflect.TypeOf(config).Elem()
	v := reflect.ValueOf(config).Elem()
	for i := 0; i < t.NumField(); i++ {
		h.Write([]byte(t.Field(i).Name))
		switch fieldValue := v.Field(i).Interface().(type) {
		case string:
			h.Write([]byte(fieldValue))
		case []byte:
			h.Write(fieldValue)
		case bool:
			var b [1]byte
			if fieldValue {
				b[0] = 1
			}
			h.Write(b[:])
		case *synapseconf.PostgresConfig:
			if fieldValue != nil {
				h.Write([]byte(fieldValue.User))
				h.Write([]byte(fieldValue.Password))
				h.Write([]byte(fieldValue.Database))
				h.Write([]byte(fieldValue.Host))
				h.Write([]byte(fieldValue.Port))
			}
		}
	}

	id = hex.EncodeToString(h.Sum(nil))
	return config, id
}

// SynapseLogConfig just returns a static string for now.
func synapseLogConfig() string {
	return `version: 1

formatters:
  precise:
   format: '%(asctime)s - %(name)s - %(lineno)d - %(levelname)s - %(request)s - %(message)s'

filters:
  context:
    (): synapse.logging.context.LoggingContextFilter
    request: ""

handlers:
  console:
    class: logging.StreamHandler
    formatter: precise
    filters: [context]

loggers:
    synapse.storage.SQL:
        # beware: increasing this to DEBUG will make synapse log sensitive
        # information such as access tokens.
        level: INFO

root:
    level: INFO
    handlers: [console]

disable_existing_loggers: false
`
}

// RandomString generates a printable random string of length n using a
// cryptographically-secure RNG.
func randomString(n int) string {
	scratch := make([]byte, (n+3)/4*3)
	if _, err := rand.Read(scratch); err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(scratch)[:n]
}
