/*
Copyright 2023 The Kubernetes Authors.

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

package v1alpha2

import (
	operatorv1 "sigs.k8s.io/cluster-api-operator/api/v1alpha3"
)

// CoreProvider conversions

func Convert_v1alpha2_CoreProvider_To_v1alpha3_CoreProvider(in *CoreProvider, out *operatorv1.CoreProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_CoreProvider_To_v1alpha2_CoreProvider(in *operatorv1.CoreProvider, out *CoreProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_CoreProviderList_To_v1alpha3_CoreProviderList(in *CoreProviderList, out *operatorv1.CoreProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.CoreProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_CoreProvider_To_v1alpha3_CoreProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_CoreProviderList_To_v1alpha2_CoreProviderList(in *operatorv1.CoreProviderList, out *CoreProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]CoreProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_CoreProvider_To_v1alpha2_CoreProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// BootstrapProvider conversions

func Convert_v1alpha2_BootstrapProvider_To_v1alpha3_BootstrapProvider(in *BootstrapProvider, out *operatorv1.BootstrapProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_BootstrapProvider_To_v1alpha2_BootstrapProvider(in *operatorv1.BootstrapProvider, out *BootstrapProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_BootstrapProviderList_To_v1alpha3_BootstrapProviderList(in *BootstrapProviderList, out *operatorv1.BootstrapProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.BootstrapProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_BootstrapProvider_To_v1alpha3_BootstrapProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_BootstrapProviderList_To_v1alpha2_BootstrapProviderList(in *operatorv1.BootstrapProviderList, out *BootstrapProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]BootstrapProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_BootstrapProvider_To_v1alpha2_BootstrapProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// ControlPlaneProvider conversions

func Convert_v1alpha2_ControlPlaneProvider_To_v1alpha3_ControlPlaneProvider(in *ControlPlaneProvider, out *operatorv1.ControlPlaneProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_ControlPlaneProvider_To_v1alpha2_ControlPlaneProvider(in *operatorv1.ControlPlaneProvider, out *ControlPlaneProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_ControlPlaneProviderList_To_v1alpha3_ControlPlaneProviderList(in *ControlPlaneProviderList, out *operatorv1.ControlPlaneProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.ControlPlaneProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_ControlPlaneProvider_To_v1alpha3_ControlPlaneProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_ControlPlaneProviderList_To_v1alpha2_ControlPlaneProviderList(in *operatorv1.ControlPlaneProviderList, out *ControlPlaneProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]ControlPlaneProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_ControlPlaneProvider_To_v1alpha2_ControlPlaneProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// InfrastructureProvider conversions

func Convert_v1alpha2_InfrastructureProvider_To_v1alpha3_InfrastructureProvider(in *InfrastructureProvider, out *operatorv1.InfrastructureProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_InfrastructureProvider_To_v1alpha2_InfrastructureProvider(in *operatorv1.InfrastructureProvider, out *InfrastructureProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_InfrastructureProviderList_To_v1alpha3_InfrastructureProviderList(in *InfrastructureProviderList, out *operatorv1.InfrastructureProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.InfrastructureProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_InfrastructureProvider_To_v1alpha3_InfrastructureProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_InfrastructureProviderList_To_v1alpha2_InfrastructureProviderList(in *operatorv1.InfrastructureProviderList, out *InfrastructureProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]InfrastructureProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_InfrastructureProvider_To_v1alpha2_InfrastructureProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// AddonProvider conversions

func Convert_v1alpha2_AddonProvider_To_v1alpha3_AddonProvider(in *AddonProvider, out *operatorv1.AddonProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_AddonProvider_To_v1alpha2_AddonProvider(in *operatorv1.AddonProvider, out *AddonProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_AddonProviderList_To_v1alpha3_AddonProviderList(in *AddonProviderList, out *operatorv1.AddonProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.AddonProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_AddonProvider_To_v1alpha3_AddonProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_AddonProviderList_To_v1alpha2_AddonProviderList(in *operatorv1.AddonProviderList, out *AddonProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]AddonProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_AddonProvider_To_v1alpha2_AddonProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// IPAMProvider conversions

func Convert_v1alpha2_IPAMProvider_To_v1alpha3_IPAMProvider(in *IPAMProvider, out *operatorv1.IPAMProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_IPAMProvider_To_v1alpha2_IPAMProvider(in *operatorv1.IPAMProvider, out *IPAMProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_IPAMProviderList_To_v1alpha3_IPAMProviderList(in *IPAMProviderList, out *operatorv1.IPAMProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.IPAMProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_IPAMProvider_To_v1alpha3_IPAMProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_IPAMProviderList_To_v1alpha2_IPAMProviderList(in *operatorv1.IPAMProviderList, out *IPAMProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]IPAMProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_IPAMProvider_To_v1alpha2_IPAMProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// RuntimeExtensionProvider conversions

func Convert_v1alpha2_RuntimeExtensionProvider_To_v1alpha3_RuntimeExtensionProvider(in *RuntimeExtensionProvider, out *operatorv1.RuntimeExtensionProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha2_ProviderSpec_To_v1alpha3_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha3_RuntimeExtensionProvider_To_v1alpha2_RuntimeExtensionProvider(in *operatorv1.RuntimeExtensionProvider, out *RuntimeExtensionProvider) error {
	out.ObjectMeta = in.ObjectMeta
	if err := Convert_v1alpha3_ProviderSpec_To_v1alpha2_ProviderSpec(&in.Spec.ProviderSpec, &out.Spec.ProviderSpec); err != nil {
		return err
	}

	if err := Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(&in.Status.ProviderStatus, &out.Status.ProviderStatus); err != nil {
		return err
	}

	return nil
}

func Convert_v1alpha2_RuntimeExtensionProviderList_To_v1alpha3_RuntimeExtensionProviderList(in *RuntimeExtensionProviderList, out *operatorv1.RuntimeExtensionProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]operatorv1.RuntimeExtensionProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha2_RuntimeExtensionProvider_To_v1alpha3_RuntimeExtensionProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

func Convert_v1alpha3_RuntimeExtensionProviderList_To_v1alpha2_RuntimeExtensionProviderList(in *operatorv1.RuntimeExtensionProviderList, out *RuntimeExtensionProviderList) error {
	out.ListMeta = in.ListMeta
	if in.Items != nil {
		out.Items = make([]RuntimeExtensionProvider, len(in.Items))
		for i := range in.Items {
			if err := Convert_v1alpha3_RuntimeExtensionProvider_To_v1alpha2_RuntimeExtensionProvider(&in.Items[i], &out.Items[i]); err != nil {
				return err
			}
		}
	}

	return nil
}

// ProviderStatus conversions

func Convert_v1alpha2_ProviderStatus_To_v1alpha3_ProviderStatus(in *ProviderStatus, out *operatorv1.ProviderStatus) error {
	out.Contract = in.Contract
	out.Conditions = in.Conditions
	out.ObservedGeneration = in.ObservedGeneration
	out.InstalledVersion = in.InstalledVersion

	return nil
}

func Convert_v1alpha3_ProviderStatus_To_v1alpha2_ProviderStatus(in *operatorv1.ProviderStatus, out *ProviderStatus) error {
	out.Contract = in.Contract
	out.Conditions = in.Conditions
	out.ObservedGeneration = in.ObservedGeneration
	out.InstalledVersion = in.InstalledVersion

	return nil
}
