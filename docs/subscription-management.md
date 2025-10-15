# Subscription Management 

## InstallPlanApproval setting

### Automatic

All subscriptions have the setting set to `Automatic` (c.f. [subscriptions.yaml](../installer/charts/tssc-subscriptions/templates/subscriptions.yaml)).

This default was chosen because:

- It allows patch versions to be automatically deployed.
- It allows patch versions to be available without having to cut a new release of the installer.
- It simplifies deployment because no job is required to automatically approve the installPlans.

However this setting means that if a product releases a patch version with a bug that causes deployment issues, the installer won't be able to deploy properly.

### Manual

It was decided to not make this setting available.

If a single subscription in a namespace is set to `Manual`, then all the subscriptions in that namespace will require their install plan to be approved (c.f. [namespace colocation](https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/operators/understanding-operators#olm-colocation-namespaces_olm-colocation)).
This means that:

- At install time, one would need to approve the installPlan. But there is no guarantee that it doesn't include unwanted upgrades to operators unrelated to the deployment.
- If the installer is run multiple times, for example in the event of adding a new dependency, then the installPlan could include updates to unwanted versions of unrelated subscriptions, which cannot be filtered out.

## Controlling subscriptions from config.yaml

Every product has 2 settings in the [config.yaml](../installer/config.yaml) to control how a product is managed:

- `enabled`: This should be set to false if the user plans to integrate with an external service. The product and its related subscription won't be installed at all.
- `properties.manageSubscription`: This should be set to false if the subscription for the product was already deployed on the cluster without the installer's help. In that case the existing subscription will be used, but the installer will configure the product. This could trigger errors if the Resources versions of the installed operator do not match the versions expected by the installer.
