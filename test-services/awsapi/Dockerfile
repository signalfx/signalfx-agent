FROM python:3.7

WORKDIR /opt

CMD ["python", "/opt"]

COPY ./requirements.txt ./requirements.txt
RUN pip install -r ./requirements.txt

COPY ./__main__.py ./__main__.py
