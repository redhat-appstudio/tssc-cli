
# Install RHTAP locally

* Copy the script `rhtap-local-install.sh` to the root director of rhtap-cli project on your computer. 
* You can modify the following values to `true` or `false` to manage which one you want to install or enable

```
export acs_install_enabled="false"
export quay_install_enabled="false"
export github_enabled="true"
export gitlab_enabled="true"
export jenkins_enabled="true"
```

* fill in the properies with proper values

Note: `update_github_app` can help you update webhook url and secret to Github APP. When the scirpt finish, what you need to do is to update callback url on Github APP page.