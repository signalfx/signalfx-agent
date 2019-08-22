ARG PYTHON_VERSION=2
FROM python:$PYTHON_VERSION

WORKDIR /signalfx
RUN git clone https://github.com/signalfx/tracing-examples

WORKDIR /signalfx/tracing-examples/signalfx-tracing/signalfx-python-tracing/django
RUN pip install -r requirements.txt && \
    sfx-py-trace-bootstrap

RUN python blog/manage.py makemigrations posts && \
    python blog/manage.py migrate && \
    python blog/manage.py shell -c "from django.contrib.auth.models import User; User.objects.create_superuser('testuser', 'testuser@example.com', 'testing123')"

CMD sfx-py-trace blog/manage.py runserver 7000
