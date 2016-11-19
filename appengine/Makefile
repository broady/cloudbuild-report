deploy:
	go test -v .
	go get -u google.golang.org/appengine/cmd/aedeploy
	aedeploy gcloud -q --project cloudbuild-report app deploy --promote

logs:
	gcloud --project cloudbuild-report app logs read --logs=stderr,stdout,request_log
