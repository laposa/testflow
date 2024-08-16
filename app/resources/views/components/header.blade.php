<header>
    <div class="header">
        <a href="/" class="logo"><img src="/images/icons/logo_check.svg">
            {{ config('app.name') ?? 'Test Portal' }}</a>

        @if ($currentUser)
            <div class="account">
                <img src="/images/icons/person.svg" class="account__image"></img>
                <span>{{ $currentUser->name }}</span>
                <a href="/auth/logout" title="Logout"><img
                        src="{{ url('/images/icons/logout.png') }}"></a>
            </div>
        @endif
    </div>
</header>
