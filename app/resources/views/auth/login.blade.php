<x-layout>
    <form method="POST" action="/auth/redirect">
        @csrf

        <button type="submit">Login with github</button>
    </form>
</x-layout>
