@php
    $hide_menu = false
@endphp
<header>
    <div class="header">
        <a href="/" class="logo">LTP<img src="/images/icons/logo_check.svg"></a>

        @if ($currentUser)
        <div class="account">
            <img src="/images/icons/person.svg" class="account__image"></img>
            <div class="account__menu">
                <span>{{ $currentUser->name }}</span>
                <ul>
                    <li><a href="#">Profile</a></li>
                    <li><a href="#">Settings</a></li>
                    <li><a href="/auth/logout">Logout</a></li>
                </ul>
            </div>
        </div>
        @endif
    </div>
    @if (!$hide_menu)
        <ul class="menu">
            <li><a href="/sessions">Sessions</a></li>
            <li><a href="/tests">Tests</a></li>
            <li><a href="/runs">Runs</a></li>
            <li><a href="/maintenance">Maintenance</a></li>

        </ul>
    @endif
</header>
