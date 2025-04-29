package githubapp

// gitHubNewAppForTmpl HTML template to POST a form to create a new GitHub App.
const gitHubNewAppForTmpl = `
<html>
<body>
  <form method="post" action="%s/settings/apps/new">
  <input type="submit" value="Create your GitHub App"></input>
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
  The GitHub App was successfully created.
  <br/>
  To complete the integration it must be installed in your GitHub organization(s).
  <br/>
  You can do it now or while the 'deploy' command is running.
  <form method="get" action="%s">
  <input type="submit" value="Install the GitHub App"></input>
  </form>
</body>
</html>
`
