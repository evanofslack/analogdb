FROM python:3.10

RUN apt-get -y update && apt-get install -y python3 python3-pip && pip3 install pipenv

RUN mkdir /app
WORKDIR /app
ADD ./Pipfile /app/Pipfile
ADD ./Pipfile.lock /app/Pipfile.lock

RUN pipenv install
RUN pipenv run python3 -m spacy download en_core_web_lg

ADD . /app/

ENTRYPOINT ["pipenv", "run", "python", "main.py"]
