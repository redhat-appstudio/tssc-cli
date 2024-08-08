package githubapp

// gitHubNewAppForTmpl HTML template to POST a form to create a new GitHub App.
const gitHubNewAppForTmpl = `
<html>
<body>
  <form method="post" action="%s/settings/apps/new">
  <input type="submit" value="Create your GitHub APP"></input>
  <input type="hidden" name="manifest" value='%s'"/>
  </form>
</body>
</html>
`

// gitHubAppSuccessfullyCreatedTmpl HTML template to inform the user that the
// GitHub App has been created and the next step.
const gitHubAppSuccessfullyCreatedTmpl = `
<html>
<body>
	GitHub App successfully created.
	Install <a href="%s">the new application</a> in your GitHub organization and continue the installation process.
</body>
</html>
`
