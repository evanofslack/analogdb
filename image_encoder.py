from typing import List

import requests
import torch
from img2vec_pytorch import Img2Vec
from PIL import Image
from qdrant_client import QdrantClient
from qdrant_client.http.models import Distance, VectorParams
from sklearn.metrics.pairwise import cosine_similarity


def image_embeddings(image: Image.Image) -> torch.Tensor:
    # Initialize Img2Vec with GPU
    img2vec = Img2Vec(cuda=False)

    # Read in an image (rgb format)
    # Get a vector from img2vec, returned as a torch FloatTensor
    vec = img2vec.get_vec(image, tensor=True)
    return vec.reshape((1, -1))


# def cosine_similarity(tensor1: torch.Tensor, tensor2: torch.Tensor) -> float:
#     cos = torch.nn.CosineSimilarity(dim=0)
#     return cos(tensor1, tensor2)


def test_image_embeddings():

    url = "https://d3i73ktnzbi69i.cloudfront.net/98fe51da-4b04-47db-b529-ce94f2c31219.jpeg"
    image = Image.open(requests.get(url, stream=True).raw)
    vec = image_embeddings(image=image)
    print(vec)
    print(type(vec))


def test_image_similarity():
    urls: List[str] = [
        "https://d3i73ktnzbi69i.cloudfront.net/7f708028-3a65-4ca1-ad85-8beb22e5a059.jpeg",
        "https://d3i73ktnzbi69i.cloudfront.net/467ac784-d985-4ea6-8040-b7ee3400a5e9.jpeg",
        "https://d3i73ktnzbi69i.cloudfront.net/01d0d68e-4224-4c4a-bcaa-546e7fc2de0a.jpeg",
        "https://d3i73ktnzbi69i.cloudfront.net/d426efe9-66e1-4b2b-b80f-f05dc991e235.jpeg",
        "https://d3i73ktnzbi69i.cloudfront.net/fb60ba2f-50af-4536-b2d5-81ad8b1dc747.png",
    ]
    images = [Image.open(requests.get(url, stream=True).raw) for url in urls]
    vecs = [image_embeddings(image) for image in images]

    target_vec = vecs[0]

    for vec in vecs:
        similarity = cosine_similarity(target_vec, vec)
        print(similarity)
        print("similar")


def init_qdrant():

    client = QdrantClient("10.33.1.142", port=6333)
    client.recreate_collection(
        collection_name="images",
        vectors_config=VectorParams(size=512, distance=Distance.DOT),
    )
    collection_info = client.get_collection(collection_name="images")


if __name__ == "__main__":
    test_image_similarity()
