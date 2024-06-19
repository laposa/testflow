<x-layout>
    <form action="/sessions" method="POST">
        @if ($errors->any())
            <div class="error-list">
                <ul>
                    @foreach ($errors->all() as $error)
                        <li>{{ $error }}</li>
                    @endforeach
                </ul>
            </div>
        @endif
        @csrf
        <x-portal-section title="Create new session" width="full">
            <div class="information">
                <input style="width:60%" type="text"
                       id="name"
                       name="name"
                       required
                       placeholder="Session name"
                       value="{{ old('name') }}"
                >
                <select>
                    <option>Prod environment</option>
                    <option>Beta environment</option>
                    <option>PreProd environment</option>
                    <option>Dev environment</option>
                </select>
            </div>
        </x-portal-section>
        <x-portal-section width="full">
            <x-test-suites.select :select="true" :tests="$tests"/>
        </x-portal-section>

        <button type="submit" class="button" >Create</button>
    </form>
</x-layout>
