# cloudbuild-report

Tools to assist running GitHub continuous builds on [Google Cloud Container Builder][https://cloud.google.com/container-builder/docs/].

## appengine

An App Engine application that updates GitHub status reports based on build reports.

This is hosted at https://cloudbuild-report.appspot-preview.com/

## client

A program that sends build reports to the App Engine application.

## cacher

A very experimental way to copy the contents of the workspace into a Docker image.

## Support

This is not an official Google product and comes with no support or guarantees.
