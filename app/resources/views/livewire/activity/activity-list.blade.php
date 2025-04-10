@php
    /** @var \Illuminate\Support\Collection $activity */
@endphp

<div class="activity-feed">
    @foreach ($activity as $entry)
        @php
            /** @var App\Models\SessionActivity $entry */
        @endphp
        <div class="activity-feed-item activity-feed--{{ $entry->type }}">
            <div class="activity-feed-header">
                @switch($entry->type)
                    @case(\App\Enums\SessionActivityType::review_requested)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="M9.879 7.519c1.171-1.025 3.071-1.025 4.242 0 1.172 1.025 1.172 2.687 0 3.712-.203.179-.43.326-.67.442-.745.361-1.45.999-1.45 1.827v.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Zm-9 5.25h.008v.008H12v-.008Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::review_approved)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::review_rejected)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round"
                                d="m9.75 9.75 4.5 4.5m0-4.5-4.5 4.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::session_created)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <circle cx="12" cy="12" r="9" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M12 8v8m4-4H8" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::run_dispatched)
                        <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
                            stroke-width="1.5" stroke="currentColor">
                            <circle cx="12" cy="12" r="9" />
                            <path stroke-linecap="round" stroke-linejoin="round" d="M10 8l6 4-6 4V8z" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::run_status_changed)
                        <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none">
                            <path
                                d="M18.4721 16.7023C17.3398 18.2608 15.6831 19.3584 13.8064 19.7934C11.9297 20.2284 9.95909 19.9716 8.25656 19.0701C6.55404 18.1687 5.23397 16.6832 4.53889 14.8865C3.84381 13.0898 3.82039 11.1027 4.47295 9.29011C5.12551 7.47756 6.41021 5.96135 8.09103 5.02005C9.77184 4.07875 11.7359 3.77558 13.6223 4.16623C15.5087 4.55689 17.1908 5.61514 18.3596 7.14656C19.5283 8.67797 20.1052 10.5797 19.9842 12.5023M19.9842 12.5023L21.4842 11.0023M19.9842 12.5023L18.4842 11.0023"
                                stroke="#000000" stroke-width="2" stroke-linecap="round"
                                stroke-linejoin="round" />
                            <path d="M12 8V12L15 15" stroke="#000000" stroke-width="2"
                                stroke-linecap="round" stroke-linejoin="round" />
                        </svg>
                    @break

                    @case(\App\Enums\SessionActivityType::manual_run_started)
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-plus"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><line x1="19" x2="19" y1="8" y2="14"/><line x1="22" x2="16" y1="11" y2="11"/></svg>
                    @break

                    @case (\App\Enums\SessionActivityType::manual_run_finished)
                        <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="lucide lucide-user-round-check"><path d="M2 21a8 8 0 0 1 13.292-6"/><circle cx="10" cy="8" r="5"/><path d="m16 19 2 2 4-4"/></svg>
                    @break

                    @case(\App\Enums\SessionActivityType::comment)
                        <span class="activity-feed-header-timestamp">
                            {{ $entry->created_at }}
                        </span>
                        <strong>{{ $entry->user->name }}</strong> commented
                    @break
                @endswitch
            </div>

            <div class="activity-feed-body">
                <div class="activity-feed-body-timestamp">{{ $entry->created_at }}</div>

                @if ($entry->type === \App\Enums\SessionActivityType::comment)
                    <x-markdown>{{ $entry->body }}</x-markdown>
                @else
                    {!! $entry->body !!}
                @endif

            </div>
        </div>
    @endforeach
</div>
