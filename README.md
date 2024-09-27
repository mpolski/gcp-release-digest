# GCP Release Digest

This application delivers timely and concise summaries of Google Cloud Platform release notes directly to your Slack channels. It achieves this by:

1. **Reading Release Notes:** The application fetches the latest Google Cloud release notes from a BigQuery dataset.

2. **Filtering by Relevance:**  You can configure the application to monitor specific release note types (e.g., BREAKING_CHANGE, FEATURE_LAUNCH) and deliver them to designated Slack channels.

3. **Summarizing with Gemini:**  Relevant release notes are sent to Vertex AI for summarization using the powerful Gemini Flash language model.

4. **Delivering to Slack:**  Concise summaries are then posted to your specified Slack channels, keeping your team informed about the GCP updates that matter most.


## Configuration

To run the app you need provide at minimum the environment variables.
Some variables are mandatory to run the app (like GCP Project, the model and the location and Cadence). Cadence specifies for how many days in the past we should read the release notes for.

Variables for Slack channels: at least one Slack channel webhook should be provided.

The purpose of GENERAL channel is to sent there all messages, or those that don't match any other channels by release note type.
To receive selected summarized messages configure relevant Slack channels with Incomoing Webhooks. Follow this Slack guide [https://api.slack.com/messaging/webhooks](https://api.slack.com/messaging/webhooks).
If one or more specific channels will be provided, the GENERAL slack channel is not mandatory and all messages not matching the other channels will be ignored.

Provide webhooks for each channel you create.
The following release note types are supported to be summarized to specific channels:


| Realease  Notes type  |    Slack channel     |
| --------------------- | ---------------------|
| Breaking Change       | BREAKING_CHANGE      |
| Deprecation           | DEPRECATION          |
| Security Bulletin     | SECURITY_BULLETIN    |
| Announcement          | SERVICE_ANNOUNCEMENT |
| Feature, Fixed, Issue, Libraries, Non-braking change   | GENERAL              |


## Local Development

1. Set the environment variables in env.vars file

NOTE: if Cadence is set to 3, release notes will be from the last 3 days however there's a 1 day delay (also consider your location) for publishing the release notes in BigQuery.

2. Read the environment variables

```
source .evn.vars
```

3. Run the function

```
FUNCTION_TARGET=digest LOCAL_ONLY=true go run -v cmd/main.go
```

4. Call the function

```
curl localhost:8080
```

## Deploy to Google Cloud Run Function

1. Set the environment variables in env.yaml file

Set the region you want to deploy to and the function's name:
```
REGION=
FUNCTION=
```

2. Deploy the function and verify:

```
gcloud functions deploy $FUNCTION --runtime go122 --trigger-http --entry-point digest --env-vars-file .env.yaml --region $REGION --no-allow-unauthenticated

gcloud functions list
```

3. Let the function create a BigQuery job and use a model hosted in Vertex AI

```
PROJECT_ID=$(gcloud config list --format='value(core.project)')
PROJECT_NO=$(gcloud projects list --filter=$PROJECT_ID --format="value(PROJECT_NUMBER)")

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$PROJECT_NO-compute@developer.gserviceaccount.com \
    --role=roles/bigquery.jobUser

gcloud projects add-iam-policy-binding $PROJECT_ID \
    --member=serviceAccount:$PROJECT_NO-compute@developer.gserviceaccount.com \
    --role=roles/aiplatform.user
```

4. Call the function
```
gcloud functions call $FUNCTION
```

## Google Cloud Scheduler - optional

Google Cloud Scheduler can invoke the function at regular intervals, e.g daily or weekly.

1. Enable Scheduler service if not running in your project:
```
gcloud services enable cloudscheduler.googleapis.com
```

2. Collect the function's URI:

Get the functions URL the scheduller will call:
```
URI=$(gcloud functions describe $FUNCTION --region $REGION --format="value(url)")
```

3. Create a service account for Cloud Scheduler and assign permissions

```
PROJECT_ID=$(gcloud config list --format='value(core.project)')
SA=queryscheduler
SA_EMAIL=$SA@$PROJECT_ID.iam.gserviceaccount.com

gcloud iam service-accounts create $SA \
  --description="Scheduler Service Account to trigger Cloud Function" \
  --display-name="Scheduler Service Account for Cloud Function"

gcloud projects add-iam-policy-binding $PROJECT_ID \
  --member serviceAccount:$SA_EMAIL \
  --role roles/cloudfunctions.invoker
```

4. Create the Scheduler Job

Next, create the Scheduler Job to trigger the function at a desired intervals.
First, set the time zone:
```
TZ="Europe/Warsaw"
```

Next, create the job:
```
gcloud scheduler jobs create http run-$FUNCTION \
  --schedule="0 8 * * *" \
  --time-zone="Europe/Warsaw" \
  --uri=$URI \
  --oidc-service-account-email=$SA_EMAIL \
  --location=$REGION
```

Verify the scheduler job is enabled:
```
gcloud scheduler jobs list --location=$REGION
```

Run the job now to test it.
```
gcloud scheduler jobs run run-$FUNCTION --location=$REGION
```


