<x-layout>
    <x-portal-section title="Create new session" width="full">
        <div class="information">
            <input style="width: 400px;" required type="text" id="sessionName" name="sessionName" required placeholder="Session name">
            <select name="assignees" id="">
                <option value="" disabled selected>Assign to</option>
                <option value="1">Norbert Laposa</option>
                <option value="2">Hugo Dvorak</option>
                <option value="3">Martin Miksovsky</option>
            </select>
        </div>
    </x-portal-section>
    <x-portal-section title="Test suites" width="full">
        <x-test-suites.list :select="true" :suites="$suites"/>
    </x-portal-section>

    <a class="button" href="/sessions/1">Create</a>
</x-layout>
