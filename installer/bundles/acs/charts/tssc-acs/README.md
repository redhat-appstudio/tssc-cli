# ACS Test

This chart deploys an ACS instance and creates an admin user.

Since it can take upward of 30m for the scanner to be ready, the deployment is tested in a separate chart (`tssc-acs-test`) to allow the install of the other products in parallel and cut down on install time.
