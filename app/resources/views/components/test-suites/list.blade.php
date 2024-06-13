@props(['select' => false, 'suites' => []])
<table @class([])>
    <thead>
    <tr>
        @if ($select)
        <th>
            <label for="selector-all" class="checkbox all">
            <input type="checkbox" id="selector-all"}>
            <span class="checkmark"></span>
            </label>
        </th>
        @endif
        <th>Test suite</th>
        <th class="center">Last result<br>Pass/Fail</th>
        <th>Tests</th>
    </tr>
    </thead>
    <tbody>
        @foreach($suites as $suite)
        <tr>
            @if ($select)
            <td>
                <label for="selector-suite-{{ $suite->id }}" class="checkbox suite">
                    <input type="checkbox" id="selector-suite-{{ $suite->id }}">
                    <span class="checkmark"></span>
                </label>
            </td>
            @endif
            <td>{{$suite->title}}</td>
            <td class="center"><span class="pass">{{$suite->passed}}</span>/<span class="fail">{{ $suite->failed }}</span></td>
            <td><span class="expand" data-target="selector-suite-tests-{{ $suite->id }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite->id }}" class="collapsible">
            <td>
                <x-tests.list :tests="$suite->tests" />
            </td>
        </tr>
        @endforeach
    </tbody>
</table>


