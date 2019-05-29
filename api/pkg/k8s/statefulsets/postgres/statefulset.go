package postgres

import (
	"github.com/okteto/app/api/pkg/model"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var devReplicas int32 = 1

//TranslateStatefulSet returns the statefulset for postgres
func TranslateStatefulSet(db *model.DB, s *model.Space) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      model.POSTGRES,
			Namespace: s.ID,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: model.POSTGRES,
			Replicas:    &devReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": model.POSTGRES,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": model.POSTGRES,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						apiv1.Container{
							Name:            model.POSTGRES,
							Image:           model.POSTGRES,
							ImagePullPolicy: apiv1.PullIfNotPresent,
							Ports: []apiv1.ContainerPort{
								apiv1.ContainerPort{
									ContainerPort: 5432,
								},
							},
							Env: []apiv1.EnvVar{
								apiv1.EnvVar{
									Name:  "POSTGRES_USER",
									Value: "okteto",
								},
								apiv1.EnvVar{
									Name:  "POSTGRES_PASSWORD",
									Value: db.Password,
								},
								apiv1.EnvVar{
									Name:  "POSTGRES_DB",
									Value: "db",
								},
								apiv1.EnvVar{
									Name:  "PGDATA",
									Value: db.GetVolumePath(),
								},
							},
							VolumeMounts: []apiv1.VolumeMount{
								apiv1.VolumeMount{
									Name:      db.GetVolumeName(),
									MountPath: db.GetVolumePath(),
									SubPath:   "postgresql-db",
								},
							},
						},
					},
					Volumes: []apiv1.Volume{
						apiv1.Volume{
							Name: db.GetVolumeName(),
							VolumeSource: apiv1.VolumeSource{
								PersistentVolumeClaim: &apiv1.PersistentVolumeClaimVolumeSource{
									ClaimName: db.GetVolumeName(),
									ReadOnly:  false,
								},
							},
						},
					},
				},
			},
		},
	}
}