# Install Artifactory(JFrog Container Registry) on openshift cluster

1. Login to OpenShift

2. Deploy Artifactory on Openshift

```
$ helm repo add jfrog https://charts.jfrog.io
$ helm repo update
$ helm upgrade --install jfrog-container-registry jfrog/artifactory-jcr  --version 107.98.11 --namespace artifactory-jcr -f values.yaml --create-namespace

$ oc -n artifactory-jcr create route edge artifactory-ui --service=jfrog-container-registry-artifactory --port=http-router
$ oc -n artifactory-jcr create route edge artifactory-docker --service=jfrog-container-registry-artifactory --port=http-artifactory
```

3. Login to Artifactory-JCR console
you can perform the following command to get console url

```
$ echo "https://$(kubectl -n artifactory-jcr get route artifactory-web -o jsonpath='{.spec.host}')"
```

The default username and password for the built-in administrator user are:
admin/password

You just need to fill in a new password, others can be skipped in the navigation windows when you first login to Artifactory-JCR console.

4. Create a Local Docker Repository

Go to `Administration` -> `Repositories` -> `Create a Repository` -> `Local`, Select Package Type `Docker`, fill in `Repository Key` field with `rhtap`, finally click button `Create Local Repository`.


5. Set Me Up - Docker Repository

Go to `Application` -> `JFrog Container Registry` -> `Artifacts`, Click the Docker Repository created at step  `4. Create a Local Docker Repository`, Click `Set Me Up` to set up a generic client
![Set Me Up](./images/set_me_up.png)

When you input JFrog account password, it generates an identity token and Docker auth contenxt, such like below 

```
{
	"auths": {
		"https://artifactory-web-artifactory-jcr.apps.rosa.xjiang0212416.jgt9.p3.openshiftapps.com" : {
			"auth": "xxxxxxxxxxxxxxx",
			"email": "youremail@email.com"
		}
	}
}
```

<span style="color:red"> Notice: Once `Set Me Up` is done , don't click it again, otherwise the token will be changed. Maybe images cannot show up on the Image Registry tab on existing RHTAP integrating with Artifactory.</span>

6. Verify pushing image to Artifactory
Perform the following command to get registry hostname.

```
$ kubectl -n artifactory-jcr get route artifactory -o jsonpath='{.spec.host}'
```

Push image to `rhtap` repository on Artifactory server

```
$ podman tag docker.io/mysql:latest <registry hostname>/rhtap/mysql:latest
$ <Copy the Docker auth content into a file, for instance auth.json>
$ podman push --authfile <registry hostname>/rhtap/mysql:latest 
```



