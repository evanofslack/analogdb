import boto3
import requests

# Uses the creds in ~/.aws/credentials
s3 = boto3.resource("s3")
bucket = "analog-photos"
url = "https://preview.redd.it/kjvwj3vfyue81.jpg?width=3732&format=pjpg&auto=webp&s=3354c8e44cbc470f53efd25bf6bda87fe2b901e6"
filename = "test.jpg"
# s3_image_filename = 'test_s3_image.png'
# internet_image_url = 'https://docs.python.org/3.7/_static/py.png'


# Do this as a quick and easy check to make sure your S3 access is OK
for b in s3.buckets.all():
    print(b)

req = requests.get(url, stream=True)
file_obj = req.raw
req_data = file_obj.read()
s3.Bucket(bucket).put_object(
    Key=filename,
    Body=req_data,
    ContentType="image/jpg",
)
print(f"success, uploaded {filename} to {bucket}")

# Given an Internet-accessible URL, download the image and upload it to S3,
# without needing to persist the image to disk locally
# req_for_image = requests.get(internet_image_url, stream=True)
# file_object_from_req = req_for_image.raw
# req_data = file_object_from_req.read()

# # Do the actual upload to s3
# s3.Bucket(bucket_name_to_upload_image_to).put_object(Key=s3_image_filename, Body=req_data)
