@props(['select' => false, 'suites' => []])
<table @class([])>
    <thead>
    <tr>
        @if ($select)
        <th>
        </th>
        @endif
        <th>Test suite</th>
        @if (!$select)
        <th class="center">Last result<br>Pass/Fail</th>
        @endif
        <th></th>
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
            <td><a href="{{$suite->url}}">{{$suite->title}}</a></td>
            @if (!$select)
            <td class="center"><span class="pass">{{$suite->passed}}</span>/<span class="fail">{{ $suite->failed }}</span></td>
            @endif
            <td><span class="expand" data-target="selector-suite-tests-{{ $suite->id }}">Expand</span></td>
        </tr>
        <tr id="selector-suite-tests-{{ $suite->id }}" class="collapsible">
            @if ($select)
            <td colspan="4">
            @else
            <td colspan="3">
            @endif
                <x-tests.list :tests="$suite->tests" :select="$select" />
            </td>
        </tr>
        @endforeach
    </tbody>
</table>


