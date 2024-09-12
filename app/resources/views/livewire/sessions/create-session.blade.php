@php
    /**
     * @var array $suites
     */
@endphp

<div>
    <form action="/sessions" method="POST">
        @csrf

        <section>
            <h2>New session</h2>
            <div class="information">
                <input style="width:60%" type="text" id="name" name="name" required
                    placeholder="Session name" value="{{ old('name') }}">
                <select name="environment">
                    @foreach (config('app.test_session_environments') as $env)
                        <option value="{{ $env }}"
                            @if (old('environment') === $env) selected @endif>{{ ucfirst($env) }}
                        </option>
                    @endforeach
                </select>
            </div>
        </section>

        <section>
            <x-test-suites.list-interactive :suites="$suites" />
        </section>

        <section>
            <button type="submit" class="button filled">Create</button>
        </section>
    </form>
</div>
