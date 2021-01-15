package api

import (
	"encoding/json"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
)

type App struct {
}

func (app *App) HandleMutate(w http.ResponseWriter, r *http.Request) {
	admissionReview := &admissionv1.AdmissionReview{}

	// read the AdmissionReview from the request json body
	err := readJSON(r, admissionReview)
	if err != nil {
		app.HandleError(w, r, err)
		return
	}

	// unmarshal the deploy from the AdmissionRequest
	deploy := &appsv1.Deployment{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, deploy); err != nil {
		app.HandleError(w, r, fmt.Errorf("unmarshal to deploy: %v", err))
		return
	}

	//add the lifecycle to the deployment
	cmd := []string{"sleep", "30"}
	deployPodSpec := deploy.Spec.Template.Spec
	podPreStop := &corev1.Lifecycle{
		PreStop: &corev1.Handler{
			Exec: &corev1.ExecAction{Command: cmd}},
	}
	lifecycle := deployPodSpec.Containers[0].Lifecycle
	if lifecycle == nil {
		deployPodSpec.Containers[0].Lifecycle = podPreStop
	} else {
		postStart := deployPodSpec.Containers[0].Lifecycle.PostStart
		preStop := deployPodSpec.Containers[0].Lifecycle.PreStop
		if postStart != nil {
			podPreStop = &corev1.Lifecycle{
				PostStart: deployPodSpec.Containers[0].Lifecycle.PostStart, PreStop: &corev1.Handler{
					Exec: &corev1.ExecAction{Command: cmd}},
			}
		}
		if preStop == nil {
			deployPodSpec.Containers[0].Lifecycle = podPreStop
		}
	}

	LifecycleBytes, err := json.Marshal(&deployPodSpec.Containers)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall Lifecycles: %v", err))
		return
	}

	//for i := 0; i < len(deployPodSpec.Containers); i++ {
	//	if deployPodSpec.Containers[i].Lifecycle.PostStart != nil {
	//		podPreStop = &corev1.Lifecycle{
	//			PostStart: deployPodSpec.Containers[i].Lifecycle.PostStart,PreStop: &corev1.Handler{
	//				Exec: &corev1.ExecAction{Command: cmd}}}
	//	}
	//	if deployPodSpec.Containers[i].Lifecycle.PreStop == nil {
	//		deployPodSpec.Containers[i].Lifecycle = podPreStop
	//	}
	//}

	//add the podAntiAffinity to the deployment

	//antiAffinity := deployPodSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	var labelSelector []metav1.LabelSelectorRequirement
	var antiAffinity []corev1.WeightedPodAffinityTerm
	//test := corev1.WeightedPodAffinityTerm{Weight: 100,PodAffinityTerm: corev1.PodAffinityTerm{LabelSelector: metav1.LabelSelector{MatchExpressions: metav1.LabelSelectorRequirement}}}
	for k, v := range deploy.Spec.Selector.MatchLabels {
		m := []string{v}
		labelSelector = append(labelSelector, metav1.LabelSelectorRequirement{k, "In", m})
	}
	antiAffinity = append(antiAffinity, corev1.WeightedPodAffinityTerm{
		Weight: 100, PodAffinityTerm: corev1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchExpressions: labelSelector}, TopologyKey: "kubernetes.io/hostname"}})
	if deployPodSpec.Affinity == nil {
		deployPodSpec.Affinity = &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{PreferredDuringSchedulingIgnoredDuringExecution: antiAffinity},
		}
	} else {
		if deployPodSpec.Affinity.PodAntiAffinity == nil {
			deployPodSpec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{PreferredDuringSchedulingIgnoredDuringExecution: antiAffinity}
		} else {
			if deployPodSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
				deployPodSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = antiAffinity
			} else {
				for i:=0;i<len(antiAffinity);i++ {
					deployPodSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(deployPodSpec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, antiAffinity[i])
				}
			}
		}
	}
	antiAffinityByte, err := json.Marshal(&deployPodSpec.Affinity)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall deploy DNSPolicy: %v", err))
		return
	}

	//add the DNSConfig to the deployment
	m := "2"
	namespace :=admissionReview.Request.Namespace
	if deployPodSpec.DNSPolicy == "ClusterFirst" {
		deployPodSpec.DNSPolicy = corev1.DNSNone
		deployPodSpec.DNSConfig = &corev1.PodDNSConfig{
			Nameservers: []string{"10.96.0.10"},
			Searches:    []string{namespace + ".svc.cluster.local", "svc.cluster.local"},
			Options: []corev1.PodDNSConfigOption{
				corev1.PodDNSConfigOption{Name: "ndots", Value: &m},
				corev1.PodDNSConfigOption{Name: "single-request-reopen"},
			},
		}
	}

	DNSPolicyBytes, err := json.Marshal(&deployPodSpec.DNSPolicy)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall deploy DNSPolicy: %v", err))
		return
	}

	DNSConfigBytes, err := json.Marshal(&deployPodSpec.DNSConfig)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall deploy DNSConfig: %v", err))
		return
	}

	// unmarshal the pod from the AdmissionRequest
	//pod := &corev1.Pod{}
	//if err := json.Unmarshal(admissionReview.Request.Object.Raw, pod); err != nil {
	//	app.HandleError(w, r, fmt.Errorf("unmarshal to pod: %v", err))
	//	return
	//}

	// add the DNSConfig to the pod
	//m := "2"
	//namespace :=admissionReview.Request.Namespace
	//pod.Spec.DNSPolicy = corev1.DNSNone
	//pod.Spec.DNSConfig = &corev1.PodDNSConfig{
	//	Nameservers: []string{"10.96.0.10"},
	//	Searches: []string{namespace+".svc.cluster.local","svc.cluster.local"},
	//	Options: []corev1.PodDNSConfigOption{
	//		corev1.PodDNSConfigOption{Name: "ndots", Value: &m},
	//		corev1.PodDNSConfigOption{Name: "single-request-reopen"},
	//	},
	//}

	// add the volume to the pod
	//pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
	//	Name: "hello-volume",
	//	VolumeSource: corev1.VolumeSource{
	//		ConfigMap: &corev1.ConfigMapVolumeSource{
	//			LocalObjectReference: corev1.LocalObjectReference{
	//				Name: "hello-configmap",
	//			},
	//		},
	//	},
	//})

	// add volume mount to all containers in the pod
	//for i := 0; i < len(pod.Spec.Containers); i++ {
	//	pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts, corev1.VolumeMount{
	//		Name:      "hello-volume",
	//		MountPath: "/etc/config",
	//	})
	//}

	//DNSPolicyBytes, err := json.Marshal(&pod.Spec.DNSPolicy)
	//if err != nil {
	//	app.HandleError(w, r, fmt.Errorf("marshall containers: %v", err))
	//	return
	//}

	//DNSConfigBytes, err := json.Marshal(&pod.Spec.DNSConfig)
	//if err != nil {
	//	app.HandleError(w, r, fmt.Errorf("marshall containers: %v", err))
	//	return
	//}

	//containersBytes, err := json.Marshal(&pod.Spec.Containers)
	//if err != nil {
	//	app.HandleError(w, r, fmt.Errorf("marshall containers: %v", err))
	//	return
	//}

	//volumesBytes, err := json.Marshal(&pod.Spec.Volumes)
	//if err != nil {
	//	app.HandleError(w, r, fmt.Errorf("marshall volumes: %v", err))
	//	return
	//}

	// build json patch
	patch := []JSONPatchEntry{
		//JSONPatchEntry{
		//	OP:    "add",
		//	Path:  "/metadata/labels/hello-added",
		//	Value: []byte(`"OK"`),
		//},
		//JSONPatchEntry{
		//	OP:    "replace",
		//	Path:  "/spec/containers",
		//	Value: containersBytes,
		//},
		//JSONPatchEntry{
		//	OP:    "replace",
		//	Path:  "/spec/volumes",
		//	Value: volumesBytes,
		//},
		//JSONPatchEntry{
		//	OP:    "replace",
		//	Path:  "/spec/dnsPolicy",
		//	Value: DNSPolicyBytes,
		//},
		//JSONPatchEntry{
		//	OP:    "replace",
		//	Path:  "/spec/dnsConfig",
		//	Value: DNSConfigBytes,
		//},
		JSONPatchEntry{
			OP:    "replace",
			Path:  "/spec/template/spec/dnsPolicy",
			Value: DNSPolicyBytes,
		},
		JSONPatchEntry{
			OP:    "replace",
			Path:  "/spec/template/spec/dnsConfig",
			Value: DNSConfigBytes,
		},
		JSONPatchEntry{
			OP:    "replace",
			Path:  "/spec/template/spec/containers",
			Value: LifecycleBytes,
		},
		JSONPatchEntry{
			OP:    "replace",
			Path:  "/spec/template/spec/affinity",
			Value: antiAffinityByte,
		},
	}

	patchBytes, err := json.Marshal(&patch)
	if err != nil {
		app.HandleError(w, r, fmt.Errorf("marshall jsonpatch: %v", err))
		return
	}

	patchType := admissionv1.PatchTypeJSONPatch

	// build admission response
	admissionResponse := &admissionv1.AdmissionResponse{
		UID:       admissionReview.Request.UID,
		Allowed:   true,
		Patch:     patchBytes,
		PatchType: &patchType,
	}

	respAdmissionReview := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: admissionResponse,
	}

	jsonOk(w, &respAdmissionReview)

}

type JSONPatchEntry struct {
	OP    string          `json:"op"`
	Path  string          `json:"path"`
	Value json.RawMessage `json:"value,omitempty"`
}