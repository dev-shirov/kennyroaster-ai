# Step 1 Authenticate Sandbox
gcloud auth login --login-config=<login-config.json>
gcloud auth application-default login --login-config=<login-config.json>

# Step 2 Setup requirements and permissions
export PROJECT_ID=<GCP-PROJECT-ID>
gcloud config set project $PROJECT_ID
gcloud config set run/region asia-southeast1
gcloud iam service-accounts create <GCP-SERVICE-ACCOUNT> --display-name="KennyRoaster AI Service Account"
gcloud projects add-iam-policy-binding $PROJECT_ID --member="serviceAccount:kennyroaster-ai-sa@$PROJECT_ID.iam.gserviceaccount.com"  --role="roles/aiplatform.user"

# asia-southeast1
gcloud storage buckets create gs://<GCP-BUCKET-FILEPATH>/ --location=asia-southeast1
gcloud storage buckets update gs://<GCP-BUCKET-NAME> --uniform-bucket-level-access

gcloud storage cp kennyroaster.pdf gs://kennyroaster-ai-filedata/
gcloud storage buckets add-iam-policy-binding gs://kennyroaster-ai-filedata \
  --member="serviceAccount:kennyroaster-ai-sa@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"

gcloud iam service-accounts add-iam-policy-binding "kennyroaster-ai-sa@$PROJECT_ID.iam.gserviceaccount.com" \
    --member="principal://iam.googleapis.com/locations/global/workforcePools/sandbox-p/subject/<YOUR-ACCOUNT-MEMBER-PROFILE>"  \
    --role="roles/iam.serviceAccountUser" --project=$PROJECT_ID

# Step 3 Enable the required APIs once per project (in the sandbox project)
gcloud services enable run.googleapis.com aiplatform.googleapis.com storage.googleapis.com

# Step 4 Create artifact registry repo
gcloud artifacts repositories create kenny-repo \
    --repository-format=docker \
    --location=asia-southeast1 \
    --description="KennyRoasterAi Agent"

#Step 5 Build and push container

# This is a required extra step if you are using podman, not really sure on docker.
gcloud auth print-access-token | podman login \
  -u oauth2accesstoken \
  --password-stdin \
  https://asia-southeast1-docker.pkg.dev

# You can use Docker, My preference is just podman but it's totally almost the same.
podman build -t asia-southeast1-docker.pkg.dev/$PROJECT_ID/kenny-repo/kennyroasterai:v1 .
podman push asia-southeast1-docker.pkg.dev/$PROJECT_ID/kenny-repo/kennyroasterai:v1

#Step 6 Deploy (choose your region that supports Gemini; e.g., us-central1 or europe-west4)
gcloud run deploy kennyroaster-ai \
  --image asia-southeast1-docker.pkg.dev/$PROJECT_ID/kenny-repo/kennyroasterai:v1 \
  --region=asia-southeast1 \
  --allow-unauthenticated \
  --service-account=kennyroaster-ai-sa@$PROJECT_ID.iam.gserviceaccount.com \
  --platform=managed \
  --set-env-vars=GOOGLE_CLOUD_PROJECT=$PROJECT_ID,VERTEX_LOCATION=global,BUCKET=kennyroaster-ai-filedata,FILE_OBJECT=kennyroaster.pdf  \
  --memory=512Mi --cpu=1 --min-instances=0 --max-instances=3
 



