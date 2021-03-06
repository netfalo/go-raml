from sanic import Blueprint
from sanic.views import HTTPMethodView
from sanic.response import text
import users_api


from oauth2_itsyouonline import oauth2_itsyouonline

users_if = Blueprint('users_if')


class usersView(HTTPMethodView):
    
    async def get(self, request):
     
        if not await oauth2_itsyouonline([]).check_token(request):
            return text('', 401)
        
        return await users_api.users_get(request)
    
    async def post(self, request):
     
        if not await oauth2_itsyouonline(["user:memberof:goraml-admin"]).check_token(request):
            return text('', 401)
        
        return await users_api.users_post(request)
    
users_if.add_route(usersView.as_view(), '/users')

class users_byusernameView(HTTPMethodView):
    
    async def get(self, request, username):
     
        if not await oauth2_itsyouonline(["user:memberof:goraml"]).check_token(request):
            return text('', 401)
        
        return await users_api.users_byUsername_get(request, username)
    
users_if.add_route(users_byusernameView.as_view(), '/users/<username>')

