/*
Copyright 2019 kubeflow.org.

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

package ksvc

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/apis/serving/v1beta1"
	"github.com/kubeflow/kfserving/pkg/reconciler/ksvc/resources/credentials/gcs"
	"github.com/kubeflow/kfserving/pkg/reconciler/ksvc/resources/credentials/s3"
	"github.com/onsi/gomega"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestS3CredentialBuilder(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	existingServiceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
		Secrets: []v1.ObjectReference{
			{
				Name:      "s3-secret",
				Namespace: "default",
			},
		},
	}
	existingS3Secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "s3-secret",
			Namespace: "default",
			Annotations: map[string]string{
				s3.KFServiceS3SecretEndpointAnnotation: "s3.aws.com",
			},
		},
		Data: map[string][]byte{
			s3.AWSAccessKeyIdName:     {},
			s3.AWSSecretAccessKeyName: {},
		},
	}
	scenarios := map[string]struct {
		serviceAccount        *v1.ServiceAccount
		inputConfiguration    *v1alpha1.Configuration
		expectedConfiguration *v1alpha1.Configuration
		shouldFail            bool
	}{
		"Build s3 secrets envs": {
			serviceAccount: existingServiceAccount,
			inputConfiguration: &v1alpha1.Configuration{
				Spec: v1alpha1.ConfigurationSpec{
					RevisionTemplate: &v1alpha1.RevisionTemplateSpec{
						Spec: v1alpha1.RevisionSpec{
							Container: &v1.Container{},
						},
					},
				},
			},
			expectedConfiguration: &v1alpha1.Configuration{
				Spec: v1alpha1.ConfigurationSpec{
					RevisionTemplate: &v1alpha1.RevisionTemplateSpec{
						Spec: v1alpha1.RevisionSpec{
							Container: &v1.Container{
								Env: []v1.EnvVar{
									{
										Name: s3.AWSAccessKeyId,
										ValueFrom: &v1.EnvVarSource{
											SecretKeyRef: &v1.SecretKeySelector{
												LocalObjectReference: v1.LocalObjectReference{
													Name: "s3-secret",
												},
												Key: s3.AWSAccessKeyIdName,
											},
										},
									},
									{
										Name: s3.AWSSecretAccessKey,
										ValueFrom: &v1.EnvVarSource{
											SecretKeyRef: &v1.SecretKeySelector{
												LocalObjectReference: v1.LocalObjectReference{
													Name: "s3-secret",
												},
												Key: s3.AWSSecretAccessKeyName,
											},
										},
									},
									{
										Name:  s3.S3Endpoint,
										Value: "s3.aws.com",
									},
									{
										Name:  s3.AWSEndpointUrl,
										Value: "https://s3.aws.com",
									},
								},
							},
						},
					},
				},
			},
			shouldFail: false,
		},
	}

	builder := NewCredentialBulder(c)
	for name, scenario := range scenarios {
		g.Expect(c.Create(context.TODO(), existingServiceAccount)).NotTo(gomega.HaveOccurred())
		g.Expect(c.Create(context.TODO(), existingS3Secret)).NotTo(gomega.HaveOccurred())

		err := builder.CreateSecretVolumeAndEnv(context.TODO(), scenario.serviceAccount.Namespace, scenario.serviceAccount.Name,
			scenario.inputConfiguration)
		if scenario.shouldFail && err == nil {
			t.Errorf("Test %q failed: returned success but expected error", name)
		}
		// Validate
		if !scenario.shouldFail {
			if err != nil {
				t.Errorf("Test %q failed: returned error: %v", name, err)
			}
			if diff := cmp.Diff(scenario.expectedConfiguration, scenario.inputConfiguration); diff != "" {
				t.Errorf("Test %q unexpected configuration spec (-want +got): %v", name, diff)
			}
		}
		g.Expect(c.Delete(context.TODO(), existingServiceAccount)).NotTo(gomega.HaveOccurred())
		g.Expect(c.Delete(context.TODO(), existingS3Secret)).NotTo(gomega.HaveOccurred())

	}
}

func TestGCSCredentialBuilder(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	existingServiceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
		Secrets: []v1.ObjectReference{
			{
				Name:      "user-gcp-sa",
				Namespace: "default",
			},
		},
	}
	existingGCSSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-gcp-sa",
			Namespace: "default",
		},
		Data: map[string][]byte{
			gcs.GCSCredentialFileName: {},
		},
	}
	scenarios := map[string]struct {
		serviceAccount        *v1.ServiceAccount
		inputConfiguration    *v1alpha1.Configuration
		expectedConfiguration *v1alpha1.Configuration
		shouldFail            bool
	}{
		"Build gcs secrets volume": {
			serviceAccount: existingServiceAccount,
			inputConfiguration: &v1alpha1.Configuration{
				Spec: v1alpha1.ConfigurationSpec{
					RevisionTemplate: &v1alpha1.RevisionTemplateSpec{
						Spec: v1alpha1.RevisionSpec{
							Container: &v1.Container{},
						},
					},
				},
			},
			expectedConfiguration: &v1alpha1.Configuration{
				Spec: v1alpha1.ConfigurationSpec{
					RevisionTemplate: &v1alpha1.RevisionTemplateSpec{
						Spec: v1alpha1.RevisionSpec{
							Container: &v1.Container{
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      gcs.GCSCredentialVolumeName,
										ReadOnly:  true,
										MountPath: gcs.GCSCredentialVolumeMountPath,
									},
								},
								Env: []v1.EnvVar{
									{
										Name:  gcs.GCSCredentialEnvKey,
										Value: gcs.GCSCredentialVolumeMountPath,
									},
								},
							},
							RevisionSpec: v1beta1.RevisionSpec{
								PodSpec: v1beta1.PodSpec{
									Volumes: []v1.Volume{
										{
											Name: gcs.GCSCredentialVolumeName,
											VolumeSource: v1.VolumeSource{
												Secret: &v1.SecretVolumeSource{
													SecretName: "user-gcp-sa",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			shouldFail: false,
		},
	}

	builder := NewCredentialBulder(c)
	for name, scenario := range scenarios {
		g.Expect(c.Create(context.TODO(), existingServiceAccount)).NotTo(gomega.HaveOccurred())
		g.Expect(c.Create(context.TODO(), existingGCSSecret)).NotTo(gomega.HaveOccurred())

		err := builder.CreateSecretVolumeAndEnv(context.TODO(), scenario.serviceAccount.Namespace, scenario.serviceAccount.Name,
			scenario.inputConfiguration)
		if scenario.shouldFail && err == nil {
			t.Errorf("Test %q failed: returned success but expected error", name)
		}
		// Validate
		if !scenario.shouldFail {
			if err != nil {
				t.Errorf("Test %q failed: returned error: %v", name, err)
			}
			if diff := cmp.Diff(scenario.expectedConfiguration, scenario.inputConfiguration); diff != "" {
				t.Errorf("Test %q unexpected configuration spec (-want +got): %v", name, diff)
			}
		}
		g.Expect(c.Delete(context.TODO(), existingServiceAccount)).NotTo(gomega.HaveOccurred())
		g.Expect(c.Delete(context.TODO(), existingGCSSecret)).NotTo(gomega.HaveOccurred())

	}
}