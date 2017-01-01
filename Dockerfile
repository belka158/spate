# Copyright (c) 2016 Matthias Neugebauer <mtneug@mailbox.org>
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM golang:1.7-alpine
MAINTAINER Matthias Neugebauer <mtneug@mailbox.org>

COPY . /go/src/github.com/mtneug/spate
RUN go install -v github.com/mtneug/spate

EXPOSE 8080
ENTRYPOINT ["spate"]
