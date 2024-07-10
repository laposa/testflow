<x-layout>
    <x-portal-section title="Recent sessions">
        <ul>
            @foreach ($sessions as $session)
                <li>
                    <a href="/sessions/{{ $session->id }}">{{ $session->name }}</a>:
                    @if ($session->runs && isset($session->runs[0]))
                        @if ($session->runs[0]->result_log)
                            <span
                                class="pass">{{ $session->runs[0]->parsedResults->getTotalPassed() }}
                                passed</span> and <span
                                class="fail">{{ $session->runs[0]->parsedResults->getTotalFailures() }}
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
    <x-portal-section title="Service status">
        <ul>
            <li>Authorised github ID: <a href="https://github.com/norbertlaposa">@norbertlaposa</a>
            </li>
            <li>Installed under organization: <a href="https://github.com/laposa">@laposa</a></li>
            <li>Available repositories linked to portal: <strong class="online">23
                    repositories</strong></li>
        </ul>
        <a href="/settings" class="button filled">Update settings</a>
    </x-portal-section>
</x-layout>
