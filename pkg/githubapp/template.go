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
	You have successfully created a new GitHub App %q, go back to the CLI to
	finish the installation:
	<pre>
		$ rhtap-installer-cli deploy
	</pre>
</body>
</html>
`
