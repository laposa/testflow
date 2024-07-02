FROM debian:bookworm

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update \
  && apt-get -y upgrade \
  && apt-get install -y apache2 php php-zip php-sqlite3 php-xml php-curl php-pgsql nodejs npm vim curl
RUN a2enmod rewrite headers

# customise Apache configuration
RUN sed -i 's/Listen 80/Listen 8080/' /etc/apache2/ports.conf \
 && sed -i 's/:80/:8080/' /etc/apache2/sites-available/000-default.conf \
 && sed -i 's/AllowOverride None/AllowOverride All/' /etc/apache2/apache2.conf \
 && sed -i 's/ServerTokens OS/ServerTokens Prod/' /etc/apache2/conf-available/security.conf

# Configure Apache DocumentRoot to point to Laravel's public directory
# and update Apache configuration files
ENV APACHE_DOCUMENT_ROOT=/var/www/html/public
RUN sed -ri -e 's!/var/www/html!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/sites-available/*.conf
RUN sed -ri -e 's!/var/www/!${APACHE_DOCUMENT_ROOT}!g' /etc/apache2/apache2.conf /etc/apache2/conf-available/*.conf

# logs should go to stdout / stderr
RUN ln -sfT /dev/stderr /var/log/apache2/error.log \
	&& ln -sfT /dev/stdout /var/log/apache2/access.log \
	&& ln -sfT /dev/stdout /var/log/apache2/other_vhosts_access.log

# add files and switch working directory
COPY . /var/www/html/
WORKDIR /var/www/html/

# install composer
RUN cd /tmp && \
    curl -sS https://getcomposer.org/installer | php && \
    mv composer.phar /usr/local/bin/composer && \
    mkdir /var/www/.composer && \
    chown -R www-data:www-data /var/www/.composer

RUN composer install --no-interaction

RUN npm install && npm run build

# fix permissions
RUN adduser www-data tty \
  && chown -R www-data /var/log/ \
  && chown -R www-data /var/run/ \
  && chown -R www-data /var/www/html/storage /var/www/html/bootstrap/cache

ENV DEBIAN_FRONTEND teletype
ENV LC_ALL=C

# allow writable directories for AppD installation
RUN chown -R www-data /etc/apache2/conf-available \
    && chown -R www-data /etc/apache2/conf-enabled

EXPOSE 8080/tcp

# switch to non-root
USER www-data

CMD ["/usr/sbin/apache2ctl", "-DFOREGROUND"]
