import re

from sanic import Sanic
from sanic.response import text

app = Sanic()


def parse_access_key_id(request):
    auth = request.headers.get("authorization")
    try:
        return re.search(r"Credential=(.*?)/", auth).group(1)
    except IndexError:
        return None


ALLOWED_ACCESS_KEY = "MY_ACCESS_KEY"


@app.route("/iam/", methods=["POST"])
async def iam(request):
    if parse_access_key_id(request) != ALLOWED_ACCESS_KEY:
        return text("Unauthorized", status=401)
    if request.form.get("Action") == "GetUser":
        print(request.headers)
        user_name = request.form.get("UserName", "Alice")
        return text(
            f"""
<GetUserResponse xmlns="https://iam.amazonaws.com/doc/2010-05-08/">
  <GetUserResult>
    <User>
      <UserId>AIDACKCEVSQ6C2EXAMPLE</UserId>
      <Path>/</Path>
      <UserName>{user_name}</UserName>
      <Arn>arn:aws:iam::0123456789:user/{user_name}</Arn>
      <CreateDate>2013-10-02T17:01:44Z</CreateDate>
      <PasswordLastUsed>2014-10-10T14:37:51Z</PasswordLastUsed>
    </User>
  </GetUserResult>
  <ResponseMetadata>
    <RequestId>7a62c49f-347e-4fc4-9331-6e8eEXAMPLE</RequestId>
  </ResponseMetadata>
</GetUserResponse>
    """,
            content_type="application/xml",
        )


@app.route("/ec2/")
async def ec2(request):
    print(f"REQUEST ({path}): {request.json}")
    return json({"hello": "world"})


@app.route("/sts//", methods=["POST"], strict_slashes=False)
async def sts(request):
    if parse_access_key_id(request) != ALLOWED_ACCESS_KEY:
        return text("Unauthorized", status=401)
    if request.form.get("Action") == "GetCallerIdentity":
        return text(
            """
<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
   <Arn>arn:aws:iam::0123456789:user/Alice</Arn>
    <UserId>AIDACKCEVSQ6C2EXAMPLE</UserId>
    <Account>0123456789</Account>
  </GetCallerIdentityResult>
  <ResponseMetadata>
    <RequestId>01234567-89ab-cdef-0123-456789abcdef</RequestId>
  </ResponseMetadata>
</GetCallerIdentityResponse>
""",
            content_type="application/xml",
        )


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
