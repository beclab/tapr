# This file is auto generated from it's template,
# see citusdata/tools/packaging_automation/templates/docker/latest/latest.tmpl.dockerfile.
FROM postgres:14.8
ARG VERSION=12.0.0
LABEL maintainer="Citus Data https://citusdata.com" \
      org.label-schema.name="Citus" \
      org.label-schema.description="Scalable PostgreSQL for multi-tenant and real-time workloads" \
      org.label-schema.url="https://www.citusdata.com" \
      org.label-schema.vcs-url="https://github.com/citusdata/citus" \
      org.label-schema.vendor="Citus Data, Inc." \
      org.label-schema.version=${VERSION} \
      org.label-schema.schema-version="1.0"

ENV CITUS_VERSION ${VERSION}.citus-1

# install Citus
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       ca-certificates \
       curl \
       lsb-release \
       postgresql-14-pgvector 

COPY docker/citus/deb.sh /tmp/deb.sh
RUN arch=$(arch | sed s/aarch64/arm64/ | sed s/x86_64/amd64/) && \
    bash /tmp/deb.sh $arch | bash \
    && apt-get install -y postgresql-$PG_MAJOR-citus-12.0=$CITUS_VERSION \
                          postgresql-$PG_MAJOR-hll=2.17.citus-1 \
                          postgresql-$PG_MAJOR-topn=2.5.0.citus-1 

RUN arch=$(arch | sed s/aarch64/arm64/ | sed s/x86_64/amd64/) && \
    curl -sfLO https://github.com/tensorchord/pgvecto.rs/releases/download/v0.3.0/vectors-pg14_0.3.0_${arch}.deb && \
    apt-get install ./vectors-pg14_0.3.0_${arch}.deb && rm vectors-pg14_0.3.0_${arch}.deb

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
       build-essential \
       postgresql-server-dev-14 \
       lbzip2 \
        && curl -sfLO http://www.xunsearch.com/scws/down/scws-1.2.3.tar.bz2  \
        && tar xf scws-1.2.3.tar.bz2 \
        && cd scws-1.2.3 \
        && ./configure \
        && make install \
        && cd .. \
        && curl -sfLO https://github.com/amutu/zhparser/archive/refs/tags/V2.2.tar.gz \
        && tar xzvf V2.2.tar.gz \
        && cd zhparser-2.2 \
        && make \
        && make install \
        && cd .. \
        && apt-get install -y locales-all \
        && locale-gen zh_CN.UTF-8 \
        && apt-get purge -y --auto-remove build-essential postgresql-server-dev-14 libpq-dev \
        && rm -rf scws-1.2.3.tar.bz2 scws-1.2.3 V2.2.tar.gz zhparser-2.2


RUN curl https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor | tee /etc/apt/trusted.gpg.d/apt.postgresql.org.gpg >/dev/null \
    && sh -c 'echo "deb http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list' \
    && apt-get update \
    && apt-get install -y postgresql-14-postgis-3 \
    && apt-get purge -y --auto-remove curl \
    && rm -rf /var/lib/apt/lists/*



# add citus to default PostgreSQL config
RUN echo "shared_preload_libraries='citus,vectors.so'" >> /usr/share/postgresql/postgresql.conf.sample

# add scripts to run after initdb
COPY docker/citus/001-create-citus-extension.sql /docker-entrypoint-initdb.d/

# add health check script
COPY docker/citus/pg_healthcheck docker/citus/wait-for-manager.sh /
RUN chmod +x /wait-for-manager.sh

# entry point unsets PGPASSWORD, but we need it to connect to workers
# https://github.com/docker-library/postgres/blob/33bccfcaddd0679f55ee1028c012d26cd196537d/12/docker-entrypoint.sh#L303
RUN sed "/unset PGPASSWORD/d" -i /usr/local/bin/docker-entrypoint.sh

HEALTHCHECK --interval=4s --start-period=6s CMD ./pg_healthcheck
