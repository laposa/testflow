<x-layout>
    <style>
        div.login {
            margin: auto;
        }
    </style>
    <div class="login">
        <p>You need to login via your GitHub account.</p>
        <form method="POST" action="/auth/redirect">
            @csrf

            <button type="submit" class="filled">Login with github</button>
        </form>
    </div>
</x-layout>
