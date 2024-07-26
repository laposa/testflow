<x-layout>
    <x-portal-section title="Recent sessions">
        <ul>
            @foreach ($sessions as $session)
                <li>
                    <a href="/sessions/{{ $session->id }}">{{ $session->name }}</a>:
                    @if ($session->runs && isset($session->runs[0]))
                        @if (!$session->isRunning)
                            <span class="pass">{{ $session->passed }}
                                passed</span> and <span class="fail">{{ $session->failed }}
                                failed</span>
                        @else
                            <span>Running</span>
                        @endif
                    @else
                        <span>No runs yet</span>
                    @endif
                </li>
            @endforeach
        </ul>
        <a href="/sessions" class="button filled">View all sessions</a>
        <a href="/sessions/create" class="button filled">Create new session</a>
    </x-portal-section>
</x-layout>
