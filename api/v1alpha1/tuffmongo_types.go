package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TuffMongoSpec defines the desired state of TuffMongo
type TuffMongoSpec struct {
	MongoReplicas      int32                  `json:"mongoReplicas,omitempty"`
	MongoImage         string                 `json:"mongoImage,omitempty"`
	MongoContainerName string                 `json:"mongoContainerName,omitEmpty"`
	MongoPorts         []corev1.ContainerPort `json:"mongoPorts,omitEmpty"`
	MongoVolumeMounts  []corev1.VolumeMount   `json:"mongoVolumeMounts,omitEmpty"`
	MongoVolumes       []corev1.Volume        `json:"mongoVolumes,omitEmpty"`
}

// TuffMongoStatus defines the observed state of TuffMongo
type TuffMongoStatus struct {
	MongoPodNames          []string `json:"mongoPodNames"`
	MongoAvailableReplicas int32    `json:"mongoAvailableReplicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TuffMongo is the Schema for the tuffmongoes API
type TuffMongo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TuffMongoSpec   `json:"spec,omitempty"`
	Status TuffMongoStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TuffMongoList contains a list of TuffMongo
type TuffMongoList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TuffMongo `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TuffMongo{}, &TuffMongoList{})
}
